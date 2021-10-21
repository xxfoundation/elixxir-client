///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	preimage2 "gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/cmix"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"strconv"
	"strings"
	"time"
)

// Interface for SendCMIX comms; allows mocking this in testing.
type sendCmixCommsInterface interface {
	// SendPutMessage places a cMix message on the gateway to be
	// sent through cMix.
	SendPutMessage(host *connect.Host, message *pb.GatewaySlot,
		timeout time.Duration) (*pb.GatewaySlotResponse, error)
	// SendPutManyMessages places a list of cMix messages on the gateway
	// to be sent through cMix.
	SendPutManyMessages(host *connect.Host, messages *pb.GatewaySlots,
		timeout time.Duration) (*pb.GatewaySlotResponse, error)
}

// how much in the future a round needs to be to send to it
const sendTimeBuffer = 1000 * time.Millisecond
const unrecoverableError = "failed with an unrecoverable error"

// handlePutMessageError handles errors received from a PutMessage or a
// PutManyMessage network call. A printable error will be returned giving more
// context. If the error is not among recoverable errors, then the recoverable
// boolean will be returned false. If the error is among recoverable errors,
// then the boolean will return true.
// recoverable means we should try resending to the round
func handlePutMessageError(firstGateway *id.ID, instance *network.Instance,
	session *storage.Session, nodeRegistration chan network.NodeGateway,
	recipientString string, bestRound *pb.RoundInfo,
	err error) (returnErr error) {

	// If the comm errors or the message fails to send, then continue retrying;
	// otherwise, return if it sends properly
	if strings.Contains(err.Error(), "try a different round.") {
		return errors.WithMessagef(err, "Failed to send to [%s] due to "+
			"round error with round %d, bailing...",
			recipientString, bestRound.ID)
	} else if strings.Contains(err.Error(), "Could not authenticate client. "+
		"Is the client registered with this node?") {
		// If send failed due to the gateway not recognizing the authorization,
		// then renegotiate with the node to refresh it
		nodeID := firstGateway.DeepCopy()
		nodeID.SetType(id.Node)

		// Delete the keys
		session.Cmix().Remove(nodeID)

		// Trigger
		go handleMissingNodeKeys(instance, nodeRegistration, []*id.ID{nodeID})

		return errors.WithMessagef(err, "Failed to send to [%s] via %s "+
			"due to failed authentication, retrying...",
			recipientString, firstGateway)
	}

	return errors.WithMessage(err, "Failed to put cmix message")

}

// processRound is a helper function that determines the gateway to send to for
// a round and retrieves the round keys.
func processRound(instance *network.Instance, session *storage.Session,
	nodeRegistration chan network.NodeGateway, bestRound *pb.RoundInfo,
	recipientString, messageDigest string) (*id.ID, *cmix.RoundKeys, error) {

	// Build the topology
	idList, err := id.NewIDListFromBytes(bestRound.Topology)
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "Failed to use topology for "+
			"round %d when sending to [%s] (msgDigest(s): %s)",
			bestRound.ID, recipientString, messageDigest)
	}
	topology := connect.NewCircuit(idList)

	// Get the keys for the round, reject if any nodes do not have keying
	// relationships
	roundKeys, missingKeys := session.Cmix().GetRoundKeys(topology)
	if len(missingKeys) > 0 {
		go handleMissingNodeKeys(instance, nodeRegistration, missingKeys)

		return nil, nil, errors.Errorf("Failed to send on round %d to [%s] "+
			"(msgDigest(s): %s) due to missing relationships with nodes: %s",
			bestRound.ID, recipientString, messageDigest, missingKeys)
	}

	// Get the gateway to transmit to
	firstGateway := topology.GetNodeAtIndex(0).DeepCopy()
	firstGateway.SetType(id.Gateway)

	return firstGateway, roundKeys, nil
}

// buildSlotMessage is a helper function which forms a slotted message to send
// to a gateway. It encrypts passed in message and generates an ephemeral ID for
// the recipient.
func buildSlotMessage(msg format.Message, recipient *id.ID, target *id.ID,
	stream *fastRNG.Stream, senderId *id.ID, bestRound *pb.RoundInfo,
	roundKeys *cmix.RoundKeys, param params.CMIX) (*pb.GatewaySlot, format.Message, ephemeral.Id,
	error) {

	// Set the ephemeral ID
	ephID, _, _, err := ephemeral.GetId(recipient,
		uint(bestRound.AddressSpaceSize),
		int64(bestRound.Timestamps[states.QUEUED]))
	if err != nil {
		jww.FATAL.Panicf("Failed to generate ephemeral ID when sending to %s "+
			"(msgDigest: %s):  %+v", err, recipient, msg.Digest())
	}

	ephIdFilled, err := ephID.Fill(uint(bestRound.AddressSpaceSize), stream)
	if err != nil {
		jww.FATAL.Panicf("Failed to obfuscate the ephemeralID when sending "+
			"to %s (msgDigest: %s): %+v", recipient, msg.Digest(), err)
	}

	msg.SetEphemeralRID(ephIdFilled[:])

	// use the alternate identity preimage if it is set
	var preimage []byte
	if param.IdentityPreimage != nil {
		preimage = param.IdentityPreimage
		jww.INFO.Printf("Sending to %s with override preimage %v", preimage)
	}else{
		preimage = preimage2.MakeDefault(recipient)
		jww.INFO.Printf("Sending to %s with default preimage %v", preimage)
	}

	// Set the identity fingerprint
	ifp := fingerprint.IdentityFP(msg.GetContents(), preimage)

	msg.SetIdentityFP(ifp)

	// Encrypt the message
	salt := make([]byte, 32)
	_, err = stream.Read(salt)
	if err != nil {
		jww.ERROR.Printf("Failed to generate salt when sending to %s "+
			"(msgDigest: %s): %+v", recipient, msg.Digest(), err)
		return nil, format.Message{}, ephemeral.Id{}, errors.WithMessage(err,
			"Failed to generate salt, this should never happen")
	}

	encMsg, kmacs := roundKeys.Encrypt(msg, salt, id.Round(bestRound.ID))

	// Build the message payload
	msgPacket := &pb.Slot{
		SenderID: senderId.Bytes(),
		PayloadA: encMsg.GetPayloadA(),
		PayloadB: encMsg.GetPayloadB(),
		Salt:     salt,
		KMACs:    kmacs,
	}

	// Create the wrapper to the gateway
	slot := &pb.GatewaySlot{
		Message: msgPacket,
		RoundID: bestRound.ID,
		Target:  target.Bytes(),
	}

	// Add the mac proving ownership
	slot.MAC = roundKeys.MakeClientGatewayKey(salt,
		network.GenerateSlotDigest(slot))

	return slot, encMsg, ephID, nil
}

// handleMissingNodeKeys signals to the node registration thread to register a
// node if keys are missing. Identity is triggered automatically when the node
// is first seen, so this should on trigger on rare events.
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

// messageMapToStrings serializes a map of IDs and messages into a string of IDs
// and a string of message digests. Intended for use in printing to logs.
func messageMapToStrings(msgList map[id.ID]format.Message) (string, string) {
	idStrings := make([]string, 0, len(msgList))
	msgDigests := make([]string, 0, len(msgList))
	for uid, msg := range msgList {
		idStrings = append(idStrings, uid.String())
		msgDigests = append(msgDigests, msg.Digest())
	}

	return strings.Join(idStrings, ","), strings.Join(msgDigests, ",")
}

// messagesToDigestString serializes a list of messages into a string of message
// digests. Intended for use in printing to the logs.
func messagesToDigestString(msgs []format.Message) string {
	msgDigests := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		msgDigests = append(msgDigests, msg.Digest())
	}

	return strings.Join(msgDigests, ",")
}

// ephemeralIdListToString serializes a list of ephemeral IDs into a human-
// readable format. Intended for use in printing to logs.
func ephemeralIdListToString(idList []ephemeral.Id) string {
	idStrings := make([]string, 0, len(idList))

	for i := 0; i < len(idList); i++ {
		ephIdStr := strconv.FormatInt(idList[i].Int64(), 10)
		idStrings = append(idStrings, ephIdStr)
	}

	return strings.Join(idStrings, ",")
}
