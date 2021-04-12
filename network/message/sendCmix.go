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
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
	"time"
)

// interface for SendCMIX comms; allows mocking this in testing
type sendCmixCommsInterface interface {
	SendPutMessage(host *connect.Host, message *pb.GatewaySlot) (*pb.GatewaySlotResponse, error)
}

// 1.5 seconds
const sendTimeBuffer = 2500 * time.Millisecond

// WARNING: Potentially Unsafe
// Public manager function to send a message over CMIX
func (m *Manager) SendCMIX(sender *gateway.Sender, msg format.Message, recipient *id.ID, param params.CMIX) (id.Round, ephemeral.Id, error) {
	msgCopy := msg.Copy()
	return sendCmixHelper(sender, msgCopy, recipient, param, m.Instance, m.Session, m.nodeRegistration, m.Rng, m.TransmissionID, m.Comms)
}

// Payloads send are not End to End encrypted, MetaData is NOT protected with
// this call, see SendE2E for End to End encryption and full privacy protection
// Internal SendCmix which bypasses the network check, will attempt to send to
// the network without checking state. It has a built in retry system which can
// be configured through the params object.
// If the message is successfully sent, the id of the round sent it is returned,
// which can be registered with the network instance to get a callback on
// its status
func sendCmixHelper(sender *gateway.Sender, msg format.Message, recipient *id.ID, param params.CMIX, instance *network.Instance,
	session *storage.Session, nodeRegistration chan network.NodeGateway, rng *fastRNG.StreamGenerator, senderId *id.ID,
	comms sendCmixCommsInterface) (id.Round, ephemeral.Id, error) {

	timeStart := netTime.Now()
	attempted := set.New()

	jww.INFO.Printf("Looking for round to send cMix message to %s "+
		"(msgDigest: %s)", recipient, msg.Digest())

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := netTime.Now().Sub(timeStart)

		if elapsed > param.Timeout {
			jww.INFO.Printf("No rounds to send to %s (msgDigest: %s) "+
				"were found before timeout %s", recipient, msg.Digest(),
				param.Timeout)
			return 0, ephemeral.Id{}, errors.New("Sending cmix message timed out")
		}
		if numRoundTries > 0 {
			jww.INFO.Printf("Attempt %d to find round to send message "+
				"to %s (msgDigest: %s)", numRoundTries+1, recipient,
				msg.Digest())
		}

		remainingTime := param.Timeout - elapsed
		//find the best round to send to, excluding attempted rounds
		bestRound, _ := instance.GetWaitingRounds().GetUpcomingRealtime(remainingTime, attempted, sendTimeBuffer)
		if bestRound == nil {
			continue
		}

		//add the round on to the list of attempted so it is not tried again
		attempted.Insert(bestRound)

		//set the ephemeral ID
		ephID, _, _, err := ephemeral.GetId(recipient,
			uint(bestRound.AddressSpaceSize),
			int64(bestRound.Timestamps[states.QUEUED]))
		if err != nil {
			jww.FATAL.Panicf("Failed to generate ephemeral ID when "+
				"sending to %s (msgDigest: %s):  %+v", err, recipient,
				msg.Digest())
		}

		stream := rng.GetStream()
		ephIdFilled, err := ephID.Fill(uint(bestRound.AddressSpaceSize), stream)
		if err != nil {
			jww.FATAL.Panicf("Failed to obfuscate the ephemeralID when "+
				"sending to %s (msgDigest: %s): %+v", recipient, msg.Digest(),
				err)
		}
		stream.Close()

		msg.SetEphemeralRID(ephIdFilled[:])

		//set the identity fingerprint
		ifp, err := fingerprint.IdentityFP(msg.GetContents(), recipient)
		if err != nil {
			jww.FATAL.Panicf("failed to generate the Identity "+
				"fingerprint due to unrecoverable error when sending to %s "+
				"(msgDigest: %s): %+v", recipient, msg.Digest(), err)
		}

		msg.SetIdentityFP(ifp)

		//build the topology
		idList, err := id.NewIDListFromBytes(bestRound.Topology)
		if err != nil {
			jww.ERROR.Printf("Failed to use topology for round %d when "+
				"sending to %s (msgDigest: %s): %+v", bestRound.ID,
				recipient, msg.Digest(), err)
			continue
		}
		topology := connect.NewCircuit(idList)
		//get they keys for the round, reject if any nodes do not have
		//keying relationships
		roundKeys, missingKeys := session.Cmix().GetRoundKeys(topology)
		if len(missingKeys) > 0 {
			jww.WARN.Printf("Failed to send on round %d to %s "+
				"(msgDigest: %s) due to missing relationships with nodes: %s",
				bestRound.ID, recipient, msg.Digest(), missingKeys)
			go handleMissingNodeKeys(instance, nodeRegistration, missingKeys)
			time.Sleep(param.RetryDelay)
			continue
		}

		//get the gateway to transmit to
		firstGateway := topology.GetNodeAtIndex(0).DeepCopy()
		firstGateway.SetType(id.Gateway)

		//encrypt the message
		stream = rng.GetStream()
		salt := make([]byte, 32)
		_, err = stream.Read(salt)
		stream.Close()

		if err != nil {
			jww.ERROR.Printf("Failed to generate salt when sending to "+
				"%s (msgDigest: %s): %+v", recipient, msg.Digest(), err)
			return 0, ephemeral.Id{}, errors.WithMessage(err,
				"Failed to generate salt, this should never happen")
		}

		encMsg, kmacs := roundKeys.Encrypt(msg, salt, id.Round(bestRound.ID))

		//build the message payload
		msgPacket := &pb.Slot{
			SenderID: senderId.Bytes(),
			PayloadA: encMsg.GetPayloadA(),
			PayloadB: encMsg.GetPayloadB(),
			Salt:     salt,
			KMACs:    kmacs,
		}

		//create the wrapper to the gateway
		wrappedMsg := &pb.GatewaySlot{
			Message: msgPacket,
			RoundID: bestRound.ID,
		}
		//Add the mac proving ownership
		wrappedMsg.MAC = roundKeys.MakeClientGatewayKey(salt,
			network.GenerateSlotDigest(wrappedMsg))

		jww.INFO.Printf("Sending to EphID %d (%s) on round %d, "+
			"(msgDigest: %s, ecrMsgDigest: %s) via gateway %s",
			ephID.Int64(), recipient, bestRound.ID, msg.Digest(),
			encMsg.Digest(), firstGateway.String())

		// Send the payload
		result, err := sender.SendToSpecific(firstGateway, func(host *connect.Host, target *id.ID) (interface{}, bool, error) {
			wrappedMsg.Target = target.Marshal()
			result, err := comms.SendPutMessage(host, wrappedMsg)
			if err != nil {
				if strings.Contains(err.Error(),
					"try a different round.") {
					jww.WARN.Printf("Failed to send to %s (msgDigest: %s) "+
						"due to round error with round %d, retrying: %+v",
						recipient, msg.Digest(), bestRound.ID, err)
					return nil, true, err
				} else if strings.Contains(err.Error(),
					"Could not authenticate client. Is the client registered "+
						"with this node?") {
					jww.WARN.Printf("Failed to send to %s (msgDigest: %s) "+
						"via %s due to failed authentication: %s",
						recipient, msg.Digest(), firstGateway.String(), err)
					//if we failed to send due to the gateway not recognizing our
					// authorization, renegotiate with the node to refresh it
					nodeID := firstGateway.DeepCopy()
					nodeID.SetType(id.Node)
					//delete the keys
					session.Cmix().Remove(nodeID)
					//trigger
					go handleMissingNodeKeys(instance, nodeRegistration, []*id.ID{nodeID})
					return nil, true, err
				}
			}
			return result, false, err
		})
		jww.FATAL.Printf("TEST500")
		gwSlotResp := result.(*pb.GatewaySlotResponse)

		//if the comm errors or the message fails to send, continue retrying.
		//return if it sends properly
		if err != nil {
			jww.ERROR.Printf("Failed to send to EphID %d (%s) on "+
				"round %d, trying a new round: %+v", ephID.Int64(), recipient,
				bestRound.ID, err)
			continue
		} else if gwSlotResp.Accepted {
			jww.INFO.Printf("Successfully sent to EphID %v (source: %s) "+
				"in round %d", ephID.Int64(), recipient, bestRound.ID)
			return id.Round(bestRound.ID), ephID, nil
		} else {
			jww.FATAL.Panicf("Gateway %s returned no error, but failed "+
				"to accept message when sending to EphID %d (%s) on round %d",
				firstGateway.String(), ephID.Int64(), recipient, bestRound.ID)
		}
	}
	return 0, ephemeral.Id{}, errors.New("failed to send the message, " +
		"unknown error")
}

// Signals to the node registration thread to register a node if keys are
// missing. Identity is triggered automatically when the node is first seen,
// so this should on trigger on rare events.
func handleMissingNodeKeys(instance *network.Instance,
	newNodeChan chan network.NodeGateway, nodes []*id.ID) {
	for _, n := range nodes {
		ng, err := instance.GetNodeAndGateway(n)
		if err != nil {
			jww.ERROR.Printf("Node contained in round cannot be found: %s", err)
			continue
		}
		select {
		case newNodeChan <- ng:
		default:
			jww.ERROR.Printf("Failed to send node registration for %s", n)
		}

	}
}
