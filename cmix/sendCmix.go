///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/nodes"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/rateLimiting"
)

// Send sends a "raw" cMix message payload to the provided recipient.
// Returns the round ID of the round the payload was sent or an error if it
// fails.
// This does not have end-to-end encryption on it and is used exclusively as
// a send for higher order cryptographic protocols. Do not use unless
// implementing a protocol on top.
//   recipient - cMix ID of the recipient.
//   fingerprint - Key Fingerprint. 256-bit field to store a 255-bit
//      fingerprint, highest order bit must be 0 (panic otherwise). If your
//      system does not use key fingerprints, this must be random bits.
//   service - Reception Service. The backup way for a client to identify
//      messages on receipt via trial hashing and to identify notifications.
//      If unused, use message.GetRandomService to fill the field with
//      random data.
//   payload - Contents of the message. Cannot exceed the payload size for a
//      cMix message (panic otherwise).
//   mac - 256-bit field to store a 255-bit mac, highest order bit must be 0
//      (panic otherwise). If used, fill with random bits.
// Will return an error if the network is unhealthy or if it fails to send
// (along with the reason). Blocks until successful sends or errors.
// WARNING: Do not roll your own crypto.
func (c *client) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte, cmixParams CMIXParams) (
	id.Round, ephemeral.Id, error) {
	// create an internal assembler function to pass to sendWithAssembler
	assembler := func(rid id.Round) (format.Fingerprint, message.Service, []byte, []byte) {
		return fingerprint, service, payload, mac
	}
	return c.sendWithAssembler(recipient, assembler, cmixParams)
}

// SendWithAssembler sends a variable cmix payload to the provided recipient.
// The payload sent is based on the Complier function passed in, which accepts
// a round ID and returns the necessary payload data.
// Returns the round ID of the round the payload was sent or an error if it
// fails.
// This does not have end-to-end encryption on it and is used exclusively as
// a send for higher order cryptographic protocols. Do not use unless
// implementing a protocol on top.
//   recipient - cMix ID of the recipient.
//   assembler - MessageAssembler function, accepting round ID and returning fingerprint
//   format.Fingerprint, service message.Service, payload, mac []byte
// Will return an error if the network is unhealthy or if it fails to send
// (along with the reason). Blocks until successful sends or errors.
// WARNING: Do not roll your own crypto.
func (c *client) SendWithAssembler(recipient *id.ID, assembler MessageAssembler, cmixParams CMIXParams) (
	id.Round, ephemeral.Id, error) {
	// Critical messaging and assembler-based message payloads are not compatible
	if cmixParams.Critical {
		return 0, ephemeral.Id{}, errors.New("Cannot send critical messages with a message assembler")
	}
	return c.sendWithAssembler(recipient, assembler, cmixParams)
}

// sendWithAssembler wraps the passed in MessageAssembler in a messageAssembler for sendCmixHelper,
// and sets up critical message handling where applicable.
func (c *client) sendWithAssembler(recipient *id.ID, assembler MessageAssembler, cmixParams CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if !c.Monitor.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(
			"Cannot send cmix message when the network is not healthy")
	}

	// Create an internal messageAssembler which returns a format.Message
	assemblerFunc := func(rid id.Round) (format.Message, error) {
		fingerprint, service, payload, mac := assembler(rid)

		if len(payload) != c.maxMsgLen {
			return format.Message{}, errors.Errorf(
				"bad message length (%d, need %d)",
				len(payload), c.maxMsgLen)
		}

		// Build message. Will panic if inputs are not correct.
		msg := format.NewMessage(c.session.GetCmixGroup().GetP().ByteLen())
		msg.SetContents(payload)
		msg.SetKeyFP(fingerprint)
		msg.SetSIH(service.Hash(msg.GetContents()))
		msg.SetMac(mac)

		jww.TRACE.Printf("sendCmix Contents: %v, KeyFP: %v, MAC: %v, SIH: %v",
			msg.GetContents(), msg.GetKeyFP(), msg.GetMac(),
			msg.GetSIH())

		if cmixParams.Critical {
			c.crit.AddProcessing(msg, recipient, cmixParams)
		}
		return msg, nil
	}

	rid, ephID, msg, rtnErr := sendCmixHelper(c.Sender, assemblerFunc, recipient, cmixParams,
		c.instance, c.session.GetCmixGroup(), c.Registrar, c.rng, c.events,
		c.session.GetTransmissionID(), c.comms)

	if cmixParams.Critical {
		c.crit.handle(msg, recipient, rid, rtnErr)
	}

	return rid, ephID, rtnErr
}

// sendCmixHelper is a helper function for client.SendCMIX.
// NOTE: Payloads sent are not end-to-end encrypted; metadata is NOT protected
// with this call. See SendE2E for end-to-end encryption and full privacy
// protection.
// Internal SendCmix, which bypasses the network check, will attempt to send to
// the network without checking state. It has a built-in retry system which can
// be configured through the params object.
// If the message is successfully sent, the ID of the round sent it is returned,
// which can be registered with the network instance to get a callback on its
// status.
func sendCmixHelper(sender gateway.Sender, assembler messageAssembler, recipient *id.ID,
	cmixParams CMIXParams, instance *network.Instance, grp *cyclic.Group,
	nodes nodes.Registrar, rng *fastRNG.StreamGenerator, events event.Reporter,
	senderId *id.ID, comms SendCmixCommsInterface) (id.Round, ephemeral.Id, format.Message, error) {

	timeStart := netTime.Now()
	maxTimeout := sender.GetHostParams().SendTimeout

	var attempted excludedRounds.ExcludedRounds
	if cmixParams.ExcludedRounds != nil {
		attempted = cmixParams.ExcludedRounds
	} else {
		attempted = excludedRounds.NewSet()
	}

	jww.INFO.Printf("[Send-%s] Looking for round to send cMix message to "+
		"%s", cmixParams.DebugTag, recipient)

	stream := rng.GetStream()
	defer stream.Close()

	for numRoundTries := uint(0); numRoundTries < cmixParams.RoundTries; numRoundTries++ {
		elapsed := netTime.Since(timeStart)
		jww.TRACE.Printf("[Send-%s] try %d, elapsed: %s",
			cmixParams.DebugTag, numRoundTries, elapsed)

		if elapsed > cmixParams.Timeout {
			jww.INFO.Printf("[Send-%s] No rounds to send to %s "+
				"were found before timeout %s",
				cmixParams.DebugTag, recipient, cmixParams.Timeout)
			return 0, ephemeral.Id{}, format.Message{}, errors.New("Sending cmix message timed out")
		}

		if numRoundTries > 0 {
			jww.INFO.Printf("[Send-%s] Attempt %d to find round to send "+
				"message to %s", cmixParams.DebugTag,
				numRoundTries+1, recipient)
		}

		// Find the best round to send to, excluding attempted rounds
		remainingTime := cmixParams.Timeout - elapsed
		bestRound, err := instance.GetWaitingRounds().GetUpcomingRealtime(
			remainingTime, attempted, sendTimeBuffer)
		if err != nil {
			jww.WARN.Printf("[Send-%s] Failed to GetUpcomingRealtime: "+
				"%+v", cmixParams.DebugTag, err)
		}

		if bestRound == nil {
			jww.WARN.Printf(
				"[Send-%s] Best round on send is nil", cmixParams.DebugTag)
			continue
		}

		jww.TRACE.Printf("[Send-%s] Best round found: %+v",
			cmixParams.DebugTag, bestRound)

		// Determine whether the selected round contains any nodes that are
		// blacklisted by the CMIXParams object
		containsBlacklisted := false
		if cmixParams.BlacklistedNodes != nil {
			for _, nodeId := range bestRound.Topology {
				var nid id.ID
				copy(nid[:], nodeId)
				if _, isBlacklisted := cmixParams.BlacklistedNodes[nid]; isBlacklisted {
					containsBlacklisted = true
					break
				}
			}
		}

		if containsBlacklisted {
			jww.WARN.Printf("[Send-%s] Round %d contains blacklisted "+
				"nodes, skipping...", cmixParams.DebugTag, bestRound.ID)
			continue
		}

		msg, err := assembler(id.Round(bestRound.ID))
		if err != nil {
			jww.ERROR.Printf("Failed to compile message: %+v", err)
			return 0, ephemeral.Id{}, format.Message{}, err
		}

		// Flip leading bits randomly to thwart a tagging attack.
		// See cmix.SetGroupBits for more info.
		cmix.SetGroupBits(msg, grp, stream)

		// Retrieve host and key information from round
		firstGateway, roundKeys, err := processRound(
			nodes, bestRound, recipient.String(), msg.Digest())
		if err != nil {
			jww.WARN.Printf("[Send-%s] SendCmix failed to process round "+
				"(will retry): %v", cmixParams.DebugTag, err)
			continue
		}

		jww.TRACE.Printf("[Send-%s] Round %v processed, firstGW: %s",
			cmixParams.DebugTag, bestRound, firstGateway)

		// Build the messages to send
		wrappedMsg, encMsg, ephID, err := buildSlotMessage(msg, recipient,
			firstGateway, stream, senderId, bestRound, roundKeys)
		if err != nil {
			return 0, ephemeral.Id{}, format.Message{}, err
		}

		jww.INFO.Printf("[Send-%s] Sending to EphID %d (%s), on round %d "+
			"(msgDigest: %s, ecrMsgDigest: %s) via gateway %s",
			cmixParams.DebugTag, ephID.Int64(), recipient, bestRound.ID,
			msg.Digest(), encMsg.Digest(), firstGateway.String())

		// Send the payload
		sendFunc := func(host *connect.Host, target *id.ID,
			timeout time.Duration) (interface{}, error) {
			wrappedMsg.Target = target.Marshal()

			jww.TRACE.Printf(
				"[Send-%s] sendFunc %s", cmixParams.DebugTag, host)

			// Use the smaller of the two timeout durations
			timeout = calculateSendTimeout(bestRound, maxTimeout)
			calculatedTimeout := calculateSendTimeout(bestRound, maxTimeout)
			if calculatedTimeout < timeout {
				timeout = calculatedTimeout
			}

			// Send the message
			result, err := comms.SendPutMessage(host, wrappedMsg, timeout)
			jww.TRACE.Printf("[Send-%s] sendFunc %s put message",
				cmixParams.DebugTag, host)

			if err != nil {
				err := handlePutMessageError(
					firstGateway, nodes, recipient.String(), bestRound, err)
				jww.TRACE.Printf("[Send-%s] sendFunc %s error: %+v",
					cmixParams.DebugTag, host, err)
				return result, errors.WithMessagef(
					err, "SendCmix %s", unrecoverableError)
			}

			return result, err
		}

		jww.TRACE.Printf("[Send-%s] sendToPreferred %s",
			cmixParams.DebugTag, firstGateway)

		result, err := sender.SendToPreferred([]*id.ID{firstGateway}, sendFunc,
			cmixParams.Stop, cmixParams.SendTimeout)
		jww.DEBUG.Printf("[Send-%s] sendToPreferred %s returned",
			cmixParams.DebugTag, firstGateway)

		// Exit if the thread has been stopped
		if stoppable.CheckErr(err) {
			return 0, ephemeral.Id{}, format.Message{}, err
		}

		// If the comm errors or the message fails to send, continue retrying
		if err != nil {
			if strings.Contains(err.Error(), rateLimiting.ClientRateLimitErr) {
				jww.ERROR.Printf("[Send-%s] SendCmix failed to send to "+
					"EphID %d (%s) on round %d: %+v", cmixParams.DebugTag,
					ephID.Int64(), recipient, bestRound.ID, err)
				return 0, ephemeral.Id{}, format.Message{}, err
			}

			jww.ERROR.Printf("[Send-%s] SendCmix failed to send to "+
				"EphID %d (%s) on round %d, trying a new round: %+v",
				cmixParams.DebugTag, ephID.Int64(), recipient, bestRound.ID, err)
			continue
		}

		// Return if it sends properly
		gwSlotResp := result.(*pb.GatewaySlotResponse)
		if gwSlotResp.Accepted {
			m := fmt.Sprintf("[Send-%s] Successfully sent to EphID %v "+
				"(source: %s) in round %d (msgDigest: %s), elapsed: %s "+
				"numRoundTries: %d", cmixParams.DebugTag, ephID.Int64(),
				recipient, bestRound.ID, msg.Digest(), elapsed, numRoundTries)

			jww.INFO.Print(m)
			events.Report(1, "MessageSend", "Metric", m)

			return id.Round(bestRound.ID), ephID, msg, nil
		} else {
			jww.FATAL.Panicf("[Send-%s] Gateway %s returned no error, "+
				"but failed to accept message when sending to EphID %d (%s) "+
				"on round %d", cmixParams.DebugTag, firstGateway, ephID.Int64(),
				recipient, bestRound.ID)
		}

	}
	return 0, ephemeral.Id{}, format.Message{},
		errors.New("failed to send the message, unknown error")
}
