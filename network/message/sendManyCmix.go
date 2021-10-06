///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
)

// SendManyCMIX sends many "raw" cMix message payloads to each of the provided
// recipients. Used to send messages in group chats. Metadata is NOT protected
// with this call and can leak data about yourself. Returns the round ID of the
// round the payload was sent or an error if it fails.
// WARNING: Potentially Unsafe
func (m *Manager) SendManyCMIX(sender *gateway.Sender,
	messages map[id.ID]format.Message, p params.CMIX) (id.Round, []ephemeral.Id,
	error) {

	// Create message copies
	messagesCopy := make(map[id.ID]format.Message, len(messages))
	for rid, msg := range messages {
		messagesCopy[rid] = msg.Copy()
	}

	return sendManyCmixHelper(sender, messagesCopy, p, m.Instance, m.Session,
		m.nodeRegistration, m.Rng, m.TransmissionID, m.Comms)
}

// sendManyCmixHelper is a helper function for Manager.SendManyCMIX.
//
// NOTE: Payloads sent are not end to end encrypted, metadata is NOT protected
// with this call; see SendE2E for end to end encryption and full privacy
// protection. Internal SendManyCMIX, which bypasses the network check, will
// attempt to send to the network without checking state. It has a built in
// retry system which can be configured through the params object.
//
// If the message is successfully sent, the ID of the round sent it is returned,
// which can be registered with the network instance to get a callback on its
// status.
func sendManyCmixHelper(sender *gateway.Sender, msgs map[id.ID]format.Message,
	param params.CMIX, instance *network.Instance, session *storage.Session,
	nodeRegistration chan network.NodeGateway, rng *fastRNG.StreamGenerator,
	senderId *id.ID, comms sendCmixCommsInterface) (id.Round, []ephemeral.Id, error) {

	timeStart := netTime.Now()
	attempted := set.New()
	stream := rng.GetStream()
	defer stream.Close()

	maxTimeout := sender.GetHostParams().SendTimeout

	recipientString, msgDigests := messageMapToStrings(msgs)

	jww.INFO.Printf("Looking for round to send cMix messages to [%s] "+
		"(msgDigest: %s)", recipientString, msgDigests)

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := netTime.Since(timeStart)

		if elapsed > param.Timeout {
			jww.INFO.Printf("No rounds to send to %s (msgDigest: %s) were found "+
				"before timeout %s", recipientString, msgDigests, param.Timeout)
			return 0, []ephemeral.Id{},
				errors.New("sending cMix message timed out")
		}

		if numRoundTries > 0 {
			jww.INFO.Printf("Attempt %d to find round to send message to %s "+
				"(msgDigest: %s)", numRoundTries+1, recipientString, msgDigests)
		}

		remainingTime := param.Timeout - elapsed

		// Find the best round to send to, excluding attempted rounds
		bestRound, _ := instance.GetWaitingRounds().GetUpcomingRealtime(
			remainingTime, attempted, sendTimeBuffer)
		if bestRound == nil {
			continue
		}

		// Add the round on to the list of attempted so it is not tried again
		attempted.Insert(bestRound)

		// Retrieve host and key information from round
		firstGateway, roundKeys, err := processRound(instance, session,
			nodeRegistration, bestRound, recipientString, msgDigests)
		if err != nil {
			jww.WARN.Printf("SendManyCMIX failed to process round %d "+
				"(will retry): %+v", bestRound.ID, err)
			continue
		}

		// Build a slot for every message and recipient
		slots := make([]*pb.GatewaySlot, len(msgs))
		ephemeralIds := make([]ephemeral.Id, len(msgs))
		encMsgs := make([]format.Message, len(msgs))
		i := 0
		for recipient, msg := range msgs {
			slots[i], encMsgs[i], ephemeralIds[i], err = buildSlotMessage(
				msg, &recipient, firstGateway, stream, senderId, bestRound, roundKeys)
			if err != nil {
				return 0, []ephemeral.Id{}, errors.Errorf("failed to build "+
					"slot message for %s: %+v", recipient, err)
			}
			i++
		}

		// Serialize lists into a printable format
		ephemeralIdsString := ephemeralIdListToString(ephemeralIds)
		encMsgsDigest := messagesToDigestString(encMsgs)

		jww.INFO.Printf("Sending to EphIDs [%s] (%s) on round %d, "+
			"(msgDigest: %s, ecrMsgDigest: %s) via gateway %s",
			ephemeralIdsString, recipientString, bestRound.ID, msgDigests,
			encMsgsDigest, firstGateway)

		// Wrap slots in the proper message type
		wrappedMessage := &pb.GatewaySlots{
			Messages: slots,
			RoundID:  bestRound.ID,
		}

		// Send the payload
		sendFunc := func(host *connect.Host, target *id.ID) (interface{}, error) {
			wrappedMessage.Target = target.Marshal()
			timeout := calculateSendTimeout(bestRound, maxTimeout)
			result, err := comms.SendPutManyMessages(host,
				wrappedMessage, timeout)
			if err != nil {
				warn, err := handlePutMessageError(firstGateway, instance,
					session, nodeRegistration, recipientString, bestRound, err)
				if warn {
					jww.WARN.Printf("SendManyCMIX Failed: %+v", err)
				} else {
					return result, errors.WithMessagef(err,
						"SendManyCMIX %s", unrecoverableError)
				}
			}
			return result, err
		}
		result, err := sender.SendToPreferred([]*id.ID{firstGateway}, sendFunc, nil)

		// If the comm errors or the message fails to send, continue retrying
		if err != nil {
			if !strings.Contains(err.Error(), unrecoverableError) {
				jww.ERROR.Printf("SendManyCMIX failed to send to EphIDs [%s] "+
					"(sources: %s) on round %d, trying a new round %+v",
					ephemeralIdsString, recipientString, bestRound.ID, err)
				continue
			}

			return 0, []ephemeral.Id{}, err
		}

		// Return if it sends properly
		gwSlotResp := result.(*pb.GatewaySlotResponse)
		if gwSlotResp.Accepted {
			jww.INFO.Printf("Successfully sent to EphIDs %v (sources: [%s]) in "+
				"round %d", ephemeralIdsString, recipientString, bestRound.ID)
			return id.Round(bestRound.ID), ephemeralIds, nil
		} else {
			jww.FATAL.Panicf("Gateway %s returned no error, but failed to "+
				"accept message when sending to EphIDs [%s] (%s) on round %d",
				firstGateway, ephemeralIdsString, recipientString, bestRound.ID)
		}
	}

	return 0, []ephemeral.Id{},
		errors.New("failed to send the message, unknown error")
}
