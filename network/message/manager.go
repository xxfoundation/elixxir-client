///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/base64"
	"fmt"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message/parse"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
)

type Manager struct {
	param       params.Network
	partitioner parse.Partitioner
	internal.Internal
	sender           *gateway.Sender
	blacklistedNodes map[string]interface{}

	messageReception chan Bundle
	nodeRegistration chan network.NodeGateway
	networkIsHealthy chan bool
	triggerGarbled   chan struct{}

	FingerprintsManager
	TriggersManager
}

func NewManager(internal internal.Internal, param params.Network,
	nodeRegistration chan network.NodeGateway, sender *gateway.Sender) *Manager {
	dummyMessage := format.NewMessage(internal.Session.Cmix().GetGroup().GetP().ByteLen())
	m := Manager{
		param:            param,
		partitioner:      parse.NewPartitioner(dummyMessage.ContentsSize(), internal.Session),
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		networkIsHealthy: make(chan bool, 1),
		triggerGarbled:   make(chan struct{}, 100),
		nodeRegistration: nodeRegistration,
		sender:           sender,
		Internal:         internal,
	}
	for _, nodeId := range param.BlacklistedNodes {
		decodedId, err := base64.StdEncoding.DecodeString(nodeId)
		if err != nil {
			jww.ERROR.Printf("Unable to decode blacklisted Node ID %s: %+v", decodedId, err)
			continue
		}
		m.blacklistedNodes[string(decodedId)] = nil
	}

	m.FingerprintsManager = *NewFingerprints()
	m.TriggersManager = *NewTriggers()
	return &m
}

//Gets the channel to send received messages on
func (m *Manager) GetMessageReceptionChannel() chan<- Bundle {
	return m.messageReception
}

//Starts all worker pool
func (m *Manager) StartProcessies() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	//create the message handler workers
	for i := uint(0); i < m.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("MessageReception Worker %v", i))
		go m.handleMessages(stop)
		multi.Add(stop)
	}

	//create the critical messages thread
	critStop := stoppable.NewSingle("CriticalMessages")
	go m.processCriticalMessages(critStop)
	m.Health.AddChannel(m.networkIsHealthy)
	multi.Add(critStop)

	//create the garbled messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go m.processGarbledMessages(garbledStop)
	multi.Add(garbledStop)

	return multi
}
