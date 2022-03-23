///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/base64"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"strconv"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
)

const (
	inProcessKey = "InProcessMessagesKey"
)

type Pickup interface {
	GetMessageReceptionChannel() chan<- Bundle
	StartProcesses() stoppable.Stoppable
	CheckInProgressMessages()

	// Fingerprints
	AddFingerprint(clientID *id.ID, fingerprint format.Fingerprint, mp interfaces.MessageProcessor) error
	DeleteFingerprint(clientID *id.ID, fingerprint format.Fingerprint)
	DeleteClientFingerprints(clientID *id.ID)

	// Triggers
	AddTrigger(clientID *id.ID, newTrigger interfaces.Trigger, response interfaces.MessageProcessor)
	DeleteTriggers(clientID *id.ID, preimage interfaces.Preimage, response interfaces.MessageProcessor) error
	DeleteClientTriggers(clientID *id.ID)
}

type pickup struct {
	param            params.Network
	blacklistedNodes map[string]interface{}

	messageReception chan Bundle
	checkInProgress  chan struct{}

	inProcess *MeteredCmixMessageBuffer

	events interfaces.EventManager

	FingerprintsManager
	TriggersManager
}

func NewPickup(param params.Network, kv *versioned.KV, events interfaces.EventManager) Pickup {

	garbled, err := NewOrLoadMeteredCmixMessageBuffer(kv, inProcessKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load or new the Garbled Messages system: %v", err)
	}

	m := pickup{
		param:            param,
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		checkInProgress:  make(chan struct{}, 100),
		inProcess:        garbled,
		events:           events,
	}
	for _, nodeId := range param.BlacklistedNodes {
		decodedId, err := base64.StdEncoding.DecodeString(nodeId)
		if err != nil {
			jww.ERROR.Printf("Unable to decode blacklisted Node ID %s: %+v",
				decodedId, err)
			continue
		}
		m.blacklistedNodes[string(decodedId)] = nil
	}

	m.FingerprintsManager = *newFingerprints()
	m.TriggersManager = *NewTriggers()
	return &m
}

// GetMessageReceptionChannel gets the channel to send received messages on.
func (p *pickup) GetMessageReceptionChannel() chan<- Bundle {
	return p.messageReception
}

// StartProcesses starts all worker pool.
func (p *pickup) StartProcesses() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	// Create the message handler workers
	for i := uint(0); i < p.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(
			"MessageReception Worker " + strconv.Itoa(int(i)))
		go p.handleMessages(stop)
		multi.Add(stop)
	}

	// Create the in progress messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go p.recheckInProgressRunner(garbledStop)
	multi.Add(garbledStop)

	return multi
}
