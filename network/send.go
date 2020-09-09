////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (m *Manager) SendE2E(msg message.Send, e2eP params.E2E, cmixP params.CMIX) (
	[]id.Round, error) {

	if !m.health.IsRunning() {
		return nil, errors.New("Cannot send e2e message when the " +
			"network is not healthy")
	}

	return nil, nil
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (m *Manager) SendUnsafe(msg message.Send) ([]id.Round, error) {
	return nil, nil
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (m *Manager) SendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {
	if !m.health.IsRunning() {
		return 0, errors.New("Cannot send cmix message when the " +
			"network is not healthy")
	}

	return m.sendCMIX(msg, param)
}

// Internal send e2e which bypasses the network check, for use in SendE2E and
// SendUnsafe which do their own network checks
func (m *Manager) sendCMIX(msg format.Message, param params.CMIX) (id.Round, error) {

	timeStart := time.Now()
	attempted := set.New()

	for numRoundTries := uint(0); numRoundTries < param.RoundTries; numRoundTries++ {
		elapsed := time.Now().Sub(timeStart)
		if elapsed < param.Timeout {
			return 0, errors.New("Sending cmix message timed out")
		}
		remainingTime := param.Timeout - elapsed

		//find the best round to send to, excluding roudn which have been attempted
		bestRound, _ := m.instance.GetWaitingRounds().GetUpcomingRealtime(remainingTime, attempted)
		topology, firstNode := buildToplogy(bestRound.Topology)

		//get they keys for the round, reject if any nodes do not have
		//keying relationships
		roundKeys, missingKeys := m.Context.Session.Cmix().GetRoundKeys(topology)
		if len(missingKeys) > 0 {
			go handleMissingNodeKeys(missingKeys)
			continue
		}

		//get the gateway to transmit to
		firstGateway := firstNode.DeepCopy()
		firstGateway.SetType(id.Gateway)
		transmitGateway, ok := m.Comms.GetHost(firstGateway)
		if !ok {
			jww.ERROR.Printf("Failed to get host for gateway %s", transmitGateway)
			continue
		}

		//cutoff

		//encrypt the message
		salt := make([]byte, 32)
		_, err := csprng.NewSystemRNG().Read(salt)
		if err != nil {
			return 0, errors.WithMessage(err, "Failed to generate "+
				"salt, this should never happen")
		}

		encMsg, kmacs := roundKeys.Encrypt(msg, salt)

		//build the message payload
		msgPacket := &mixmessages.Slot{
			SenderID: m.uid.Bytes(),
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

func buildToplogy(nodes [][]byte) (*connect.Circuit, *id.ID) {
	idList := make([]*id.ID, len(nodes))
	for i, n := range nodes {

	}

}

func handleMissingNodeKeys(nodes []*id.ID) {}
