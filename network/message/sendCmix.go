///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"strings"
	"time"
)

// WARNING: Potentially Unsafe
// Public manager function to send a message over CMIX
func (m *Manager) SendCMIX(sender *gateway.Sender, msg format.Message,
	recipient *id.ID, cmixParams params.CMIX,
	stop *stoppable.Single) (id.Round, ephemeral.Id, error) {

	msgCopy := msg.Copy()
	return sendCmixHelper(sender, msgCopy, recipient, cmixParams, m.blacklistedNodes, m.Instance,
		m.Session, m.nodeRegistration, m.Rng, m.Internal.Events,
		m.TransmissionID, m.Comms, stop)
}

func calculateSendTimeout(best *pb.RoundInfo, max time.Duration) time.Duration {
	RoundStartTime := time.Unix(0, int64(best.Timestamps[states.QUEUED]))
	// 250ms AFTER the round starts to hear the response.
	timeout := RoundStartTime.Sub(netTime.Now().Add(250 * time.Millisecond))
	if timeout > max {
		timeout = max
	}
	// time.Duration is a signed int, so check for negative
	if timeout < 0 {
		// TODO: should this produce a warning?
		timeout = 100 * time.Millisecond
	}
	return timeout
}

// Helper function for sendCmix
// NOTE: Payloads send are not End to End encrypted, MetaData is NOT protected with
// this call, see SendE2E for End to End encryption and full privacy protection
// Internal SendCmix which bypasses the network check, will attempt to send to
// the network without checking state. It has a built in retry system which can
// be configured through the params object.
// If the message is successfully sent, the id of the round sent it is returned,
// which can be registered with the network instance to get a callback on
// its status
func sendCmixHelper(sender *gateway.Sender, msg format.Message,
	recipient *id.ID, cmixParams params.CMIX, blacklistedNodes map[string]interface{}, instance *network.Instance,
	session *storage.Session, nodeRegistration chan network.NodeGateway,
	rng *fastRNG.StreamGenerator, events interfaces.EventManager,
	senderId *id.ID, comms sendCmixCommsInterface,
	stop *stoppable.Single) (id.Round, ephemeral.Id, error) {

	timeStart := netTime.Now()
	maxTimeout := sender.GetHostParams().SendTimeout

	var attempted excludedRounds.ExcludedRounds
	if cmixParams.ExcludedRounds != nil {
		attempted = cmixParams.ExcludedRounds
	} else {
		attempted = excludedRounds.NewSet()
	}

	jww.INFO.Printf("[SendCMIX-%s]Looking for round to send cMix message to %s "+
		"(msgDigest: %s)", cmixParams.DebugTag, recipient, msg.Digest())

	stream := rng.GetStream()
	defer stream.Close()
	grp := session.Cmix().GetGroup()

	// flip leading bits randomly to thwart a tagging attack.
	// See SetGroupBits for more info
	cmix.SetGroupBits(msg, grp, stream)

	for numRoundTries := uint(0); numRoundTries < cmixParams.RoundTries; numRoundTries++ {
		elapsed := netTime.Since(timeStart)
		jww.TRACE.Printf("[SendCMIX-%s] try %d, elapsed: %s",
			cmixParams.DebugTag, numRoundTries, elapsed)

		if elapsed > cmixParams.Timeout {
			jww.INFO.Printf("[SendCMIX-%s] No rounds to send to %s (msgDigest: %s) "+
				"were found before timeout %s", cmixParams.DebugTag, recipient, msg.Digest(),
				cmixParams.Timeout)
			return 0, ephemeral.Id{}, errors.New("Sending cmix message timed out")
		}
		if numRoundTries > 0 {
			jww.INFO.Printf("[SendCMIX-%s] Attempt %d to find round to send message "+
				"to %s (msgDigest: %s)", cmixParams.DebugTag, numRoundTries+1, recipient,
				msg.Digest())
		}

		// find the best round to send to, excluding attempted rounds
		remainingTime := cmixParams.Timeout - elapsed
		bestRound, err := instance.GetWaitingRounds().GetUpcomingRealtime(remainingTime, attempted, sendTimeBuffer)
		if err != nil {
			jww.WARN.Printf("[SendCMIX-%s] Failed to GetUpcomingRealtime "+
				"(msgDigest: %s): %+v", cmixParams.DebugTag, msg.Digest(), err)
		}
		if bestRound == nil {
			jww.WARN.Printf("[SendCMIX-%s], Best round on send is nil", cmixParams.DebugTag)
			continue
		}
		jww.TRACE.Printf("[SendCMIX-%s] bestRound: %v", cmixParams.DebugTag, bestRound)

		// Determine whether the selected round contains any Nodes
		// that are blacklisted by the params.Network object
		containsBlacklisted := false
		for _, nodeId := range bestRound.Topology {
			if _, isBlacklisted := blacklistedNodes[string(nodeId)]; isBlacklisted {
				containsBlacklisted = true
				break
			}
		}
		if containsBlacklisted {
			jww.WARN.Printf("[SendCMIX-%s]Round %d contains blacklisted node, "+
				"skipping...", cmixParams.DebugTag, bestRound.ID)
			continue
		}

		// Retrieve host and key information from round
		firstGateway, roundKeys, err := processRound(instance, session, nodeRegistration, bestRound, recipient.String(), msg.Digest())
		if err != nil {
			jww.WARN.Printf("[SendCMIX-%s]SendCmix failed to process round"+
				" (will retry): %v", cmixParams.DebugTag, err)
			continue
		}

		jww.TRACE.Printf("[SendCMIX-%s]round %v processed, firstGW: %s",
			cmixParams.DebugTag, bestRound, firstGateway)

		// Build the messages to send

		wrappedMsg, encMsg, ephID, err := buildSlotMessage(msg, recipient,
			firstGateway, stream, senderId, bestRound, roundKeys, cmixParams)
		if err != nil {
			return 0, ephemeral.Id{}, err
		}

		jww.INFO.Printf("[SendCMIX-%s] Sending to EphID %d (%s), "+
			"on round %d (msgDigest: %s, ecrMsgDigest: %s) "+
			"via gateway %s", cmixParams.DebugTag,
			ephID.Int64(), recipient, bestRound.ID, msg.Digest(),
			encMsg.Digest(), firstGateway.String())

		// Send the payload
		sendFunc := func(host *connect.Host, target *id.ID,
			timeout time.Duration) (interface{}, error) {
			wrappedMsg.Target = target.Marshal()

			jww.TRACE.Printf("[SendCMIX-%s]sendFunc %s", cmixParams.DebugTag, host)
			timeout = calculateSendTimeout(bestRound, maxTimeout)
			// Use the smaller of the two timeout durations
			calculatedTimeout := calculateSendTimeout(bestRound, maxTimeout)
			if calculatedTimeout < timeout {
				timeout = calculatedTimeout
			}

			// send the message
			result, err := comms.SendPutMessage(host, wrappedMsg,
				timeout)
			jww.TRACE.Printf("[SendCMIX-%s]sendFunc %s putmsg", cmixParams.DebugTag, host)

			if err != nil {
				// fixme: should we provide as a slice the whole topology?
				err := handlePutMessageError(firstGateway,
					instance, session, nodeRegistration,
					recipient.String(), bestRound, err)
				jww.TRACE.Printf("[SendCMIX-%s] sendFunc %s err %+v",
					cmixParams.DebugTag, host, err)
				return result, errors.WithMessagef(err,
					"SendCmix %s", unrecoverableError)
			}
			return result, err
		}
		jww.TRACE.Printf("[SendCMIX-%s] sendToPreferred %s",
			cmixParams.DebugTag, firstGateway)
		result, err := sender.SendToPreferred(
			[]*id.ID{firstGateway}, sendFunc, stop, cmixParams.SendTimeout)
		jww.DEBUG.Printf("[SendCMIX-%s] sendToPreferred %s returned",
			cmixParams.DebugTag, firstGateway)

		// Exit if the thread has been stopped
		if stoppable.CheckErr(err) {
			return 0, ephemeral.Id{}, err
		}

		// if the comm errors or the message fails to send, continue retrying.
		if err != nil {
			if strings.Contains(err.Error(), rateLimiting.ClientRateLimitErr) {
				jww.ERROR.Printf("[SendCMIX-%s] SendCmix failed to send to EphID %d (%s) on "+
					"round %d: %+v", cmixParams.DebugTag, ephID.Int64(), recipient,
					bestRound.ID, err)
				return 0, ephemeral.Id{}, err
			}

			jww.ERROR.Printf("[SendCMIX-%s] SendCmix failed to send to EphID %d (%s) on "+
				"round %d, trying a new round: %+v", cmixParams.DebugTag, ephID.Int64(), recipient,
				bestRound.ID, err)
			continue
		}

		// Return if it sends properly
		gwSlotResp := result.(*pb.GatewaySlotResponse)
		if gwSlotResp.Accepted {
			m := fmt.Sprintf("[SendCMIX-%s] Successfully sent to EphID %v "+
				"(source: %s) in round %d (msgDigest: %s), "+
				"elapsed: %s numRoundTries: %d", cmixParams.DebugTag, ephID.Int64(),
				recipient, bestRound.ID, msg.Digest(),
				elapsed, numRoundTries)
			jww.INFO.Print(m)
			events.Report(1, "MessageSend", "Metric", m)
			onSend(1, session)
			return id.Round(bestRound.ID), ephID, nil
		} else {
			jww.FATAL.Panicf("[SendCMIX-%s] Gateway %s returned no error, but failed "+
				"to accept message when sending to EphID %d (%s) on round %d",
				cmixParams.DebugTag, firstGateway, ephID.Int64(), recipient, bestRound.ID)
		}

	}
	return 0, ephemeral.Id{}, errors.New("failed to send the message, " +
		"unknown error")
}

// OnSend performs a bucket addition on a call to Manager.SendCMIX or
// Manager.SendManyCMIX, updating the bucket for the amount of messages sent.
func onSend(messages uint32, session *storage.Session) {
	rateLimitingParam := session.GetBucketParams().Get()
	session.GetBucket().AddWithExternalParams(messages,
		rateLimitingParam.Capacity, rateLimitingParam.LeakedTokens,
		rateLimitingParam.LeakDuration)

}
