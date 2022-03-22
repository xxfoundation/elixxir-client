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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/fastRNG"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
)

const (
	garbledMessagesKey = "GarbledMessages"
)

type Manager struct {
	param            params.Network
	sender           *gateway.Sender
	blacklistedNodes map[string]interface{}

	messageReception chan Bundle
	nodeRegistration chan network.NodeGateway
	networkIsHealthy chan bool
	triggerGarbled   chan struct{}

	garbledStore *utility.MeteredCmixMessageBuffer

	rng     *fastRNG.StreamGenerator
	events  interfaces.EventManager
	comms   SendCmixCommsInterface
	session *storage.Session

	FingerprintsManager
	TriggersManager
}

func NewManager(param params.Network,
	nodeRegistration chan network.NodeGateway, sender *gateway.Sender,
	session *storage.Session, rng *fastRNG.StreamGenerator,
	events interfaces.EventManager, comms SendCmixCommsInterface) *Manager {

	garbled, err := utility.NewOrLoadMeteredCmixMessageBuffer(session.GetKV(), garbledMessagesKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load or new the Garbled Messages system")
	}

	m := Manager{
		param:            param,
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		networkIsHealthy: make(chan bool, 1),
		triggerGarbled:   make(chan struct{}, 100),
		nodeRegistration: nodeRegistration,
		sender:           sender,
		garbledStore:     garbled,
		rng:              rng,
		events:           events,
		comms:            comms,
		session:          session,
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

	//create the garbled messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go m.processGarbledMessages(garbledStop)
	multi.Add(garbledStop)

	return multi
}
