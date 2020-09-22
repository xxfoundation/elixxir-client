package message

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)


// Internal send e2e which bypasses the network check, for use in SendE2E and
// SendUnsafe which do their own network checks
func (m *Manager) SendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {

	timeStart := time.Now()
	attempted := set.New()

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := time.Now().Sub(timeStart)
		if elapsed < param.Timeout {
			return 0, errors.New("Sending cmix message timed out")
		}
		remainingTime := param.Timeout - elapsed

		//find the best round to send to, excluding roudn which have been attempted
		bestRound, _ := m.Instance.GetWaitingRounds().GetUpcomingRealtime(remainingTime, attempted)
		topology, err := buildToplogy(bestRound.Topology)
		if err == nil {
			jww.ERROR.Printf("Failed to use topology for round %v: %s", bestRound.ID, err)
			continue
		}

		//get they keys for the round, reject if any nodes do not have
		//keying relationships
		roundKeys, missingKeys := m.Session.Cmix().GetRoundKeys(topology)
		if len(missingKeys) > 0 {
			go handleMissingNodeKeys(m.Instance, m.nodeRegistration, missingKeys)
			continue
		}

		//get the gateway to transmit to
		firstGateway := topology.GetNodeAtIndex(0).DeepCopy()
		firstGateway.SetType(id.Gateway)
		transmitGateway, ok := m.Comms.GetHost(firstGateway)
		if !ok {
			jww.ERROR.Printf("Failed to get host for gateway %s", transmitGateway)
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

		//Send the payload
		gwSlotResp, err := m.Comms.SendPutMessage(transmitGateway, msg)
		//if the comm errors or the message fails to send, continue retrying.
		//return if it sends properly
		if err != nil {
			jww.ERROR.Printf("Failed to send message to %s: %s",
				transmitGateway, err)
		} else if gwSlotResp.Accepted {
			return id.Round(bestRound.ID), nil
		}

		//add the round on to the list of attempted so it is not tried again
		attempted.Insert(bestRound)
	}

	return 0, errors.New("failed to send the message")
}

func buildToplogy(nodes [][]byte) (*connect.Circuit, error) {
	idList := make([]*id.ID, len(nodes))
	for i, n := range nodes {
		nid, err := id.Unmarshal(n)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to "+
				"convert topology on node %v/%v {raw id: %v}", i, len(nodes), n)
		}
		idList[i] = nid
	}
	topology := connect.NewCircuit(idList)
	return topology, nil

}

func handleMissingNodeKeys(instance *network.Instance, newNodeChan chan network.NodeGateway, nodes []*id.ID) {
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
