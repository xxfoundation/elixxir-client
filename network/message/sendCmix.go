package message

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"time"
)

const sendTimeBuffer = uint64(100*time.Millisecond)

// WARNING: Potentially Unsafe
// Payloads send are not End to End encrypted, MetaData is NOT protected with
// this call, see SendE2E for End to End encryption and full privacy protection
// Internal SendCmix which bypasses the network check, will attempt to send to
// the network without checking state. It has a built in retry system which can
// be configured through the params object.
// If the message is successfully sent, the id of the round sent it is returned,
// which can be registered with the network instance to get a callback on
// its status
func (m *Manager) SendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {

	timeStart := time.Now()
	attempted := set.New()

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := time.Now().Sub(timeStart)
		jww.DEBUG.Printf("SendCMIX Send Attempt %d", numRoundTries+1)
		if elapsed > param.Timeout {
			return 0, errors.New("Sending cmix message timed out")
		}
		remainingTime := param.Timeout - elapsed
		jww.TRACE.Printf("SendCMIX GetUpcommingRealtime")
		//find the best round to send to, excluding attempted rounds
		bestRound, _ := m.Instance.GetWaitingRounds().GetUpcomingRealtime(remainingTime, attempted)

		if (bestRound.Timestamps[states.REALTIME]+sendTimeBuffer)>
			uint64(time.Now().UnixNano()){
			jww.WARN.Println("Round received which has already started" +
				" realtime")
			continue
		}

		//build the topology
		idList, err := id.NewIDListFromBytes(bestRound.Topology)
		if err != nil {
			jww.ERROR.Printf("Failed to use topology for round %v: %s", bestRound.ID, err)
			continue
		}
		topology := connect.NewCircuit(idList)
		jww.TRACE.Printf("SendCMIX GetRoundKeys")
		//get they keys for the round, reject if any nodes do not have
		//keying relationships
		roundKeys, missingKeys := m.Session.Cmix().GetRoundKeys(topology)
		if len(missingKeys) > 0 {
			go handleMissingNodeKeys(m.Instance, m.nodeRegistration, missingKeys)
			time.Sleep(param.RetryDelay)
			continue
		}

		//get the gateway to transmit to
		firstGateway := topology.GetNodeAtIndex(0).DeepCopy()
		firstGateway.SetType(id.Gateway)

		transmitGateway, ok := m.Comms.GetHost(firstGateway)
		if !ok {
			jww.ERROR.Printf("Failed to get host for gateway %s", transmitGateway)
			time.Sleep(param.RetryDelay)
			continue
		}

		//encrypt the message
		salt := make([]byte, 32)
		stream := m.Rng.GetStream()
		_, err = stream.Read(salt)
		stream.Close()

		if err != nil {
			return 0, errors.WithMessage(err, "Failed to generate "+
				"salt, this should never happen")
		}
		jww.INFO.Printf("RECIPIENTIDPRE_ENCRYPT: %s", msg.GetRecipientID())
		encMsg, kmacs := roundKeys.Encrypt(msg, salt)

		//build the message payload
		msgPacket := &mixmessages.Slot{
			SenderID: m.Uid.Bytes(),
			PayloadA: encMsg.GetPayloadA(),
			PayloadB: encMsg.GetPayloadB(),
			Salt:     salt,
			KMACs:    kmacs,
		}

		//create the wrapper to the gateway
		msg := &mixmessages.GatewaySlot{
			Message: msgPacket,
			RoundID: bestRound.ID,
		}
		//Add the mac proving ownership
		msg.MAC = roundKeys.MakeClientGatewayKey(salt, network.GenerateSlotDigest(msg))

		//add the round on to the list of attempted so it is not tried again
		attempted.Insert(bestRound)
		jww.DEBUG.Printf("SendCMIX SendPutMessage")
		//Send the payload
		gwSlotResp, err := m.Comms.SendPutMessage(transmitGateway, msg)
		//if the comm errors or the message fails to send, continue retrying.
		//return if it sends properly
		if err != nil {
			if strings.Contains(err.Error(),
				"try a different round.") {
				jww.WARN.Printf("could not send: %s",
					err)
				continue
			}
			jww.ERROR.Printf("Failed to send message to %s: %s",
				transmitGateway, err)
		} else if gwSlotResp.Accepted {
			return id.Round(bestRound.ID), nil
		}
	}

	return 0, errors.New("failed to send the message")
}

// Signals to the node registration thread to register a node if keys are
// missing. Registration is triggered automatically when the node is first seen,
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
