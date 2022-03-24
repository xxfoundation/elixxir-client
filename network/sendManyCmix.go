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
	"strings"
	"time"
)

// TargetedCmixMessage defines a recipient target pair in a sendMany cMix
// message.
type TargetedCmixMessage struct {
	Recipient *id.ID
	Message   format.Message
}

// SendManyCMIX sends many "raw" cMix message payloads to each of the provided
// recipients. Used to send messages in group chats. Metadata is NOT protected
// with this call and can leak data about yourself. Returns the round ID of the
// round the payload was sent or an error if it fails.
// WARNING: Potentially Unsafe
func (m *manager) SendManyCMIX(messages []TargetedCmixMessage,
	p CMIXParams) (id.Round, []ephemeral.Id, error) {
	if !m.Monitor.IsHealthy() {
		return 0, []ephemeral.Id{}, errors.New("Cannot send cmix " +
			"message when the network is not healthy")
	}

	return sendManyCmixHelper(m.Sender, messages, p,
		m.instance, m.session.GetCmixGroup(), m.Registrar, m.rng, m.events,
		m.session.GetTransmissionID(), m.comms)
}

// sendManyCmixHelper is a helper function for manager.SendManyCMIX.
//
// NOTE: Payloads sent are not end-to-end encrypted, metadata is NOT protected
// with this call; see SendE2E for end-to-end encryption and full privacy
// protection. Internal SendManyCMIX, which bypasses the network check, will
// attempt to send to the network without checking state. It has a built-in
// retry system which can be configured through the params object.
//
// If the message is successfully sent, the ID of the round sent it is returned,
// which can be registered with the network instance to get a callback on its
// status.
func sendManyCmixHelper(sender gateway.Sender,
	msgs []TargetedCmixMessage, param CMIXParams, instance *network.Instance,
	grp *cyclic.Group, registrar nodes.Registrar,
	rng *fastRNG.StreamGenerator, events event.Manager,
	senderId *id.ID, comms SendCmixCommsInterface) (
	id.Round, []ephemeral.Id, error) {

	timeStart := netTime.Now()
	var attempted excludedRounds.ExcludedRounds
	if param.ExcludedRounds != nil {
		attempted = param.ExcludedRounds
	} else {
		attempted = excludedRounds.NewSet()
	}

	maxTimeout := sender.GetHostParams().SendTimeout

	recipientString, msgDigests := messageListToStrings(msgs)

	jww.INFO.Printf("[SendManyCMIX-%s]Looking for round to send cMix messages to [%s] "+
		"(msgDigest: %s)", param.DebugTag, recipientString, msgDigests)

	stream := rng.GetStream()
	defer stream.Close()

	// flip leading bits randomly to thwart a tagging attack.
	// See SetGroupBits for more info
	for i := range msgs {
		cmix.SetGroupBits(msgs[i].Message, grp, stream)
	}

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := netTime.Since(timeStart)

		if elapsed > param.Timeout {
			jww.INFO.Printf("[SendManyCMIX-%s]No rounds to send to %s (msgDigest: %s) were found "+
				"before timeout %s", param.DebugTag, recipientString, msgDigests, param.Timeout)
			return 0, []ephemeral.Id{},
				errors.New("sending cMix message timed out")
		}

		if numRoundTries > 0 {
			jww.INFO.Printf("[SendManyCMIX-%s]Attempt %d to find round to send message to %s "+
				"(msgDigest: %s)", param.DebugTag, numRoundTries+1, recipientString, msgDigests)
		}

		remainingTime := param.Timeout - elapsed

		// Find the best round to send to, excluding attempted rounds
		bestRound, _ := instance.GetWaitingRounds().GetUpcomingRealtime(
			remainingTime, attempted, sendTimeBuffer)
		if bestRound == nil {
			continue
		}

		// Determine whether the selected round contains any nodes that are
		// blacklisted by the params.Network object
		containsBlacklisted := false
		if param.BlacklistedNodes != nil {
			for _, nodeId := range bestRound.Topology {
				nid := &id.ID{}
				copy(nid[:], nodeId)
				if _, isBlacklisted := param.BlacklistedNodes[*nid]; isBlacklisted {
					containsBlacklisted = true
					break
				}
			}
		}
		if containsBlacklisted {
			jww.WARN.Printf("[SendManyCMIX-%s]Round %d contains blacklisted nodes, skipping...",
				param.DebugTag, bestRound.ID)
			continue
		}

		// Retrieve host and key information from round
		firstGateway, roundKeys, err := processRound(
			registrar, bestRound, recipientString, msgDigests)
		if err != nil {
			jww.INFO.Printf("[SendManyCMIX-%s]error processing round: %v", param.DebugTag, err)
			jww.WARN.Printf("[SendManyCMIX-%s]SendManyCMIX failed to process round %d "+
				"(will retry): %+v", param.DebugTag, bestRound.ID, err)
			continue
		}

		// Build a slot for every message and recipient
		slots := make([]*pb.GatewaySlot, len(msgs))
		encMsgs := make([]format.Message, len(msgs))
		ephemeralIDs := make([]ephemeral.Id, len(msgs))
		stream := rng.GetStream()
		for i, msg := range msgs {
			slots[i], encMsgs[i], ephemeralIDs[i], err = buildSlotMessage(
				msg.Message, msg.Recipient, firstGateway, stream, senderId,
				bestRound, roundKeys, param)
			if err != nil {
				stream.Close()
				jww.INFO.Printf("[SendManyCMIX-%s]error building slot received: %v", param.DebugTag, err)
				return 0, []ephemeral.Id{}, errors.Errorf("failed to build "+
					"slot message for %s: %+v", msg.Recipient, err)
			}
		}

		stream.Close()

		// Serialize lists into a printable format
		ephemeralIDsString := ephemeralIdListToString(ephemeralIDs)
		encMsgsDigest := messagesToDigestString(encMsgs)

		jww.INFO.Printf("[SendManyCMIX-%s]Sending to EphIDs [%s] (%s) on round %d, "+
			"(msgDigest: %s, ecrMsgDigest: %s) via gateway %s", param.DebugTag,
			ephemeralIDsString, recipientString, bestRound.ID, msgDigests,
			encMsgsDigest, firstGateway)

		// Wrap slots in the proper message type
		wrappedMessage := &pb.GatewaySlots{
			Messages: slots,
			RoundID:  bestRound.ID,
		}

		// Send the payload
		sendFunc := func(host *connect.Host, target *id.ID,
			timeout time.Duration) (interface{}, error) {
			// Use the smaller of the two timeout durations
			calculatedTimeout := calculateSendTimeout(bestRound, maxTimeout)
			if calculatedTimeout < timeout {
				timeout = calculatedTimeout
			}

			wrappedMessage.Target = target.Marshal()
			result, err := comms.SendPutManyMessages(
				host, wrappedMessage, timeout)
			if err != nil {
				err := handlePutMessageError(firstGateway, registrar,
					recipientString, bestRound, err)
				return result, errors.WithMessagef(err,
					"SendManyCMIX %s (via %s): %s",
					target, host, unrecoverableError)

			}
			return result, err
		}
		result, err := sender.SendToPreferred(
			[]*id.ID{firstGateway}, sendFunc, param.Stop, param.SendTimeout)

		// Exit if the thread has been stopped
		if stoppable.CheckErr(err) {
			return 0, []ephemeral.Id{}, err
		}

		// If the comm errors or the message fails to send, continue retrying
		if err != nil {
			if !strings.Contains(err.Error(), unrecoverableError) {
				jww.ERROR.Printf("[SendManyCMIX-%s]SendManyCMIX failed to send to EphIDs [%s] "+
					"(sources: %s) on round %d, trying a new round %+v",
					param.DebugTag, ephemeralIDsString, recipientString, bestRound.ID, err)
				jww.INFO.Printf("[SendManyCMIX-%s]error received, continuing: %v", param.DebugTag, err)
				continue
			} else {
				jww.INFO.Printf("[SendManyCMIX-%s]Error received: %v", param.DebugTag, err)
			}
			return 0, []ephemeral.Id{}, err
		}

		// Return if it sends properly
		gwSlotResp := result.(*pb.GatewaySlotResponse)
		if gwSlotResp.Accepted {
			m := fmt.Sprintf("[SendManyCMIX-%s]Successfully sent to EphIDs %s (sources: [%s]) "+
				"in round %d (msgDigest: %s)", param.DebugTag, ephemeralIDsString, recipientString, bestRound.ID, msgDigests)
			jww.INFO.Print(m)
			events.Report(1, "MessageSendMany", "Metric", m)
			return id.Round(bestRound.ID), ephemeralIDs, nil
		} else {
			jww.FATAL.Panicf("[SendManyCMIX-%s]Gateway %s returned no error, but failed to "+
				"accept message when sending to EphIDs [%s] (%s) on round %d", param.DebugTag,
				firstGateway, ephemeralIDsString, recipientString, bestRound.ID)
		}
	}

	return 0, []ephemeral.Id{},
		errors.New("failed to send the message, unknown error")
}
