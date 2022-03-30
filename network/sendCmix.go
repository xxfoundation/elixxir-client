///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/nodes"
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
	"strings"
	"time"
)

// SendCMIX sends a "raw" cMix message payload to the provided recipient.
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
func (m *manager) SendCMIX(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte, cmixParams CMIXParams) (
	id.Round, ephemeral.Id, error) {
	if !m.Monitor.IsHealthy() {
		return 0, ephemeral.Id{}, errors.New(
			"Cannot send cmix message when the network is not healthy")
	}

	// Build message. Will panic if inputs are not correct.
	msg := format.NewMessage(m.session.GetCmixGroup().GetP().ByteLen())
	msg.SetKeyFP(fingerprint)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetSIH(service.Hash(msg.GetContents()))

	if cmixParams.Critical {
		m.crit.AddProcessing(msg, recipient, cmixParams)
	}

	rid, ephID, rtnErr := sendCmixHelper(m.Sender, msg, recipient, cmixParams,
		m.instance, m.session.GetCmixGroup(), m.Registrar, m.rng, m.events,
		m.session.GetTransmissionID(), m.comms)

	if cmixParams.Critical {
		m.crit.handle(msg, recipient, rid, rtnErr)
	}

	return rid, ephID, rtnErr
}

// sendCmixHelper is a helper function for manager.SendCMIX.
// NOTE: Payloads sent are not end-to-end encrypted; metadata is NOT protected
// with this call. See SendE2E for end-to-end encryption and full privacy
// protection.
// Internal SendCmix, which bypasses the network check, will attempt to send to
// the network without checking state. It has a built-in retry system which can
// be configured through the params object.
// If the message is successfully sent, the ID of the round sent it is returned,
// which can be registered with the network instance to get a callback on its
// status.
func sendCmixHelper(sender gateway.Sender, msg format.Message, recipient *id.ID,
	cmixParams CMIXParams, instance *network.Instance, grp *cyclic.Group,
	nodes nodes.Registrar, rng *fastRNG.StreamGenerator, events event.Manager,
	senderId *id.ID, comms SendCmixCommsInterface) (id.Round, ephemeral.Id, error) {

	timeStart := netTime.Now()
	maxTimeout := sender.GetHostParams().SendTimeout

	var attempted excludedRounds.ExcludedRounds
	if cmixParams.ExcludedRounds != nil {
		attempted = cmixParams.ExcludedRounds
	} else {
		attempted = excludedRounds.NewSet()
	}

	jww.INFO.Printf("[SendCMIX-%s] Looking for round to send cMix message to "+
		"%s (msgDigest: %s)", cmixParams.DebugTag, recipient, msg.Digest())

	stream := rng.GetStream()
	defer stream.Close()

	// Flip leading bits randomly to thwart a tagging attack.
	// See cmix.SetGroupBits for more info.
	cmix.SetGroupBits(msg, grp, stream)

	for numRoundTries := uint(0); numRoundTries < cmixParams.RoundTries; numRoundTries++ {
		elapsed := netTime.Since(timeStart)
		jww.TRACE.Printf("[SendCMIX-%s] try %d, elapsed: %s",
			cmixParams.DebugTag, numRoundTries, elapsed)

		if elapsed > cmixParams.Timeout {
			jww.INFO.Printf("[SendCMIX-%s] No rounds to send to %s "+
				"(msgDigest: %s) were found before timeout %s",
				cmixParams.DebugTag, recipient, msg.Digest(), cmixParams.Timeout)
			return 0, ephemeral.Id{}, errors.New("Sending cmix message timed out")
		}

		if numRoundTries > 0 {
			jww.INFO.Printf("[SendCMIX-%s] Attempt %d to find round to send "+
				"message to %s (msgDigest: %s)", cmixParams.DebugTag,
				numRoundTries+1, recipient, msg.Digest())
		}

		// Find the best round to send to, excluding attempted rounds
		remainingTime := cmixParams.Timeout - elapsed
		bestRound, err := instance.GetWaitingRounds().GetUpcomingRealtime(
			remainingTime, attempted, sendTimeBuffer)
		if err != nil {
			jww.WARN.Printf("[SendCMIX-%s] Failed to GetUpcomingRealtime "+
				"(msgDigest: %s): %+v", cmixParams.DebugTag, msg.Digest(), err)
		}

		if bestRound == nil {
			jww.WARN.Printf(
				"[SendCMIX-%s] Best round on send is nil", cmixParams.DebugTag)
			continue
		}

		jww.TRACE.Printf("[SendCMIX-%s] Best round found: %+v",
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
			jww.WARN.Printf("[SendCMIX-%s] Round %d contains blacklisted "+
				"nodes, skipping...", cmixParams.DebugTag, bestRound.ID)
			continue
		}

		// Retrieve host and key information from round
		firstGateway, roundKeys, err := processRound(
			nodes, bestRound, recipient.String(), msg.Digest())
		if err != nil {
			jww.WARN.Printf("[SendCMIX-%s] SendCmix failed to process round "+
				"(will retry): %v", cmixParams.DebugTag, err)
			continue
		}

		jww.TRACE.Printf("[SendCMIX-%s] Round %v processed, firstGW: %s",
			cmixParams.DebugTag, bestRound, firstGateway)

		// Build the messages to send
		wrappedMsg, encMsg, ephID, err := buildSlotMessage(msg, recipient,
			firstGateway, stream, senderId, bestRound, roundKeys)
		if err != nil {
			return 0, ephemeral.Id{}, err
		}

		jww.INFO.Printf("[SendCMIX-%s] Sending to EphID %d (%s), on round %d "+
			"(msgDigest: %s, ecrMsgDigest: %s) via gateway %s",
			cmixParams.DebugTag, ephID.Int64(), recipient, bestRound.ID,
			msg.Digest(), encMsg.Digest(), firstGateway.String())

		// Send the payload
		sendFunc := func(host *connect.Host, target *id.ID,
			timeout time.Duration) (interface{}, error) {
			wrappedMsg.Target = target.Marshal()

			jww.TRACE.Printf(
				"[SendCMIX-%s] sendFunc %s", cmixParams.DebugTag, host)

			// Use the smaller of the two timeout durations
			timeout = calculateSendTimeout(bestRound, maxTimeout)
			calculatedTimeout := calculateSendTimeout(bestRound, maxTimeout)
			if calculatedTimeout < timeout {
				timeout = calculatedTimeout
			}

			// Send the message
			result, err := comms.SendPutMessage(host, wrappedMsg, timeout)
			jww.TRACE.Printf("[SendCMIX-%s] sendFunc %s put message",
				cmixParams.DebugTag, host)

			if err != nil {
				err := handlePutMessageError(
					firstGateway, nodes, recipient.String(), bestRound, err)
				jww.TRACE.Printf("[SendCMIX-%s] sendFunc %s error: %+v",
					cmixParams.DebugTag, host, err)
				return result, errors.WithMessagef(
					err, "SendCmix %s", unrecoverableError)
			}

			return result, err
		}

		jww.TRACE.Printf("[SendCMIX-%s] sendToPreferred %s",
			cmixParams.DebugTag, firstGateway)

		result, err := sender.SendToPreferred([]*id.ID{firstGateway}, sendFunc,
			cmixParams.Stop, cmixParams.SendTimeout)
		jww.DEBUG.Printf("[SendCMIX-%s] sendToPreferred %s returned",
			cmixParams.DebugTag, firstGateway)

		// Exit if the thread has been stopped
		if stoppable.CheckErr(err) {
			return 0, ephemeral.Id{}, err
		}

		// If the comm errors or the message fails to send, continue retrying
		if err != nil {
			if strings.Contains(err.Error(), rateLimiting.ClientRateLimitErr) {
				jww.ERROR.Printf("[SendCMIX-%s] SendCmix failed to send to "+
					"EphID %d (%s) on round %d: %+v", cmixParams.DebugTag,
					ephID.Int64(), recipient, bestRound.ID, err)
				return 0, ephemeral.Id{}, err
			}

			jww.ERROR.Printf("[SendCMIX-%s] SendCmix failed to send to "+
				"EphID %d (%s) on round %d, trying a new round: %+v",
				cmixParams.DebugTag, ephID.Int64(), recipient, bestRound.ID, err)
			continue
		}

		// Return if it sends properly
		gwSlotResp := result.(*pb.GatewaySlotResponse)
		if gwSlotResp.Accepted {
			m := fmt.Sprintf("[SendCMIX-%s] Successfully sent to EphID %v "+
				"(source: %s) in round %d (msgDigest: %s), elapsed: %s "+
				"numRoundTries: %d", cmixParams.DebugTag, ephID.Int64(),
				recipient, bestRound.ID, msg.Digest(), elapsed, numRoundTries)

			jww.INFO.Print(m)
			events.Report(1, "MessageSend", "Metric", m)

			return id.Round(bestRound.ID), ephID, nil
		} else {
			jww.FATAL.Panicf("[SendCMIX-%s] Gateway %s returned no error, "+
				"but failed to accept message when sending to EphID %d (%s) "+
				"on round %d", cmixParams.DebugTag, firstGateway, ephID.Int64(),
				recipient, bestRound.ID)
		}

	}
	return 0, ephemeral.Id{},
		errors.New("failed to send the message, unknown error")
}
