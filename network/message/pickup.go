///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"strconv"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
)

const (
	inProcessKey = "InProcessMessagesKey"
)

type Handler interface {
	GetMessageReceptionChannel() chan<- Bundle
	StartProcesses() stoppable.Stoppable
	CheckInProgressMessages()

	// Fingerprints
	AddFingerprint(clientID *id.ID, fingerprint format.Fingerprint, mp Processor) error
	DeleteFingerprint(clientID *id.ID, fingerprint format.Fingerprint)
	DeleteClientFingerprints(clientID *id.ID)

	// Triggers
	AddService(clientID *id.ID, newService Service, response Processor)
	DeleteService(clientID *id.ID, toDelete Service, response Processor)
	DeleteClientService(clientID *id.ID)
	TrackServices(triggerTracker ServicesTracker)
}

type handler struct {
	param Params

	messageReception chan Bundle
	checkInProgress  chan struct{}

	inProcess *MeteredCmixMessageBuffer

	events event.Manager

	FingerprintsManager
	ServicesManager
}

func NewHandler(param Params, kv *versioned.KV, events event.Manager) Handler {

	garbled, err := NewOrLoadMeteredCmixMessageBuffer(kv, inProcessKey)
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to load or new the Garbled Messages system: %v", err)
	}

	m := handler{
		param:            param,
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		checkInProgress:  make(chan struct{}, 100),
		inProcess:        garbled,
		events:           events,
	}

	m.FingerprintsManager = *newFingerprints()
	m.ServicesManager = *NewServices()
	return &m
}

// GetMessageReceptionChannel gets the channel to send received messages on.
func (p *handler) GetMessageReceptionChannel() chan<- Bundle {
	return p.messageReception
}

// StartProcesses starts all worker pool.
func (p *handler) StartProcesses() stoppable.Stoppable {
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
