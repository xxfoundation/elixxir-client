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
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

const (
	inProcessKey = "InProcessMessagesKey"
)

type Pickup interface {
	GetMessageReceptionChannel() chan<- Bundle
	StartProcessies() stoppable.Stoppable
	CheckInProgressMessages()

	//Fingerprints
	AddFingerprint(clientID *id.ID, fingerprint format.Fingerprint, mp interfaces.MessageProcessor) error
	DeleteFingerprint(clientID *id.ID, fingerprint format.Fingerprint)
	DeleteClientFingerprints(clientID *id.ID)

	//Triggers
	AddTrigger(clientID *id.ID, newTrigger interfaces.Trigger, response interfaces.MessageProcessor)
	DeleteTriggers(clientID *id.ID, preimage interfaces.Preimage, response interfaces.MessageProcessor) error
	DeleteClientTriggers(clientID *id.ID)
}

type pickup struct {
	param            params.Network
	sender           *gateway.Sender
	blacklistedNodes map[string]interface{}

	messageReception chan Bundle
	checkInProgress  chan struct{}

	inProcess *MeteredCmixMessageBuffer

	events interfaces.EventManager

	FingerprintsManager
	TriggersManager
}

func NewPickup(param params.Network, sender *gateway.Sender,
	session *storage.Session, events interfaces.EventManager) Pickup {

	garbled, err := NewOrLoadMeteredCmixMessageBuffer(session.GetKV(), inProcessKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load or new the Garbled Messages system")
	}

	m := pickup{
		param:            param,
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		checkInProgress:  make(chan struct{}, 100),
		sender:           sender,
		inProcess:        garbled,
		events:           events,
	}
	for _, nodeId := range param.BlacklistedNodes {
		decodedId, err := base64.StdEncoding.DecodeString(nodeId)
		if err != nil {
			jww.ERROR.Printf("Unable to decode blacklisted Node ID %s: %+v", decodedId, err)
			continue
		}
		m.blacklistedNodes[string(decodedId)] = nil
	}

	m.FingerprintsManager = *newFingerprints()
	m.TriggersManager = *NewTriggers()
	return &m
}

//Gets the channel to send received messages on
func (p *pickup) GetMessageReceptionChannel() chan<- Bundle {
	return p.messageReception
}

//Starts all worker pool
func (p *pickup) StartProcessies() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	//create the message handler workers
	for i := uint(0); i < p.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("MessageReception Worker %v", i))
		go p.handleMessages(stop)
		multi.Add(stop)
	}

	//create the in progress messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go p.recheckInProgressRunner(garbledStop)
	multi.Add(garbledStop)

	return multi
}
