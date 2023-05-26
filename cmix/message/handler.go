////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"sync"
	"time"

	"gitlab.com/elixxir/client/v4/event"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/id"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/netTime"
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

	// Services
	AddService(clientID *id.ID, newService Service, response Processor)
	DeleteService(clientID *id.ID, toDelete Service, response Processor)
	UpsertCompressedService(clientID *id.ID, newService CompressedService,
		response Processor)
	DeleteCompressedService(clientID *id.ID, toDelete CompressedService,
		processor Processor)
	DeleteClientService(clientID *id.ID)
	TrackServices(triggerTracker ServicesTracker)
	GetServices() (ServiceList, CompressedServiceList)

	//Fallthrough
	AddFallthrough(c *id.ID, p Processor)
	RemoveFallthrough(c *id.ID)
}

type handler struct {
	param Params

	messageReception chan Bundle
	checkInProgress  chan struct{}

	inProcess *MeteredCmixMessageBuffer

	events event.Reporter

	FingerprintsManager
	ServicesManager
	FallthroughManager
}

func NewHandler(param Params, kv versioned.KV, events event.Reporter,
	standardID *id.ID) Handler {

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

	m.FingerprintsManager = *newFingerprints(standardID)
	m.ServicesManager = *NewServices()
	m.FallthroughManager = newFallthroughManager()
	return &m
}

// GetMessageReceptionChannel gets the channel to send received messages on.
func (h *handler) GetMessageReceptionChannel() chan<- Bundle {
	return h.messageReception
}

// StartProcesses starts all worker pool.
func (h *handler) StartProcesses() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	// Create the message handler workers
	for i := uint(0); i < h.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(
			"MessageReception Worker " + strconv.Itoa(int(i)))
		go h.handleMessages(stop)
		multi.Add(stop)
	}

	// Create the in progress messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go h.recheckInProgressRunner(garbledStop)
	multi.Add(garbledStop)

	return multi
}

// handleMessages is a long-running thread that receives each Bundle from messageReception
// and processes the messages in the Bundle
func (h *handler) handleMessages(stop *stoppable.Single) {
	for {
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		case bundle := <-h.messageReception:
			go func() {
				wg := sync.WaitGroup{}
				wg.Add(len(bundle.Messages))
				for i := range bundle.Messages {
					msg := bundle.Messages[i]
					jww.TRACE.Printf("handle IterMsgs: %s",
						msg.Digest())

					go func() {
						count, ts := h.inProcess.Add(
							msg, bundle.RoundInfo.Raw, bundle.Identity)
						wg.Done()
						h.handleMessage(count, ts, msg, bundle)
					}()
				}
				wg.Wait()
				bundle.Finish()
			}()
		}
	}

}

// handleMessage processes an individual message in the Bundle
// and handles the inProcess logic
func (h *handler) handleMessage(count uint, ts time.Time, msg format.Message, bundle Bundle) {
	success := h.handleMessageHelper(msg, bundle)
	if success {
		h.inProcess.Remove(
			msg, bundle.RoundInfo.Raw, bundle.Identity)
	} else {
		// Fail the message if any part of the decryption
		// fails, unless it is the last attempts and has
		// been in the buffer long enough, in which case
		// remove it
		if count == h.param.MaxChecksInProcessMessage &&
			netTime.Since(ts) > h.param.InProcessMessageWait {
			h.inProcess.Remove(
				msg, bundle.RoundInfo.Raw, bundle.Identity)
		} else {
			h.inProcess.Failed(
				msg, bundle.RoundInfo.Raw, bundle.Identity)
		}
	}
}

// handleMessageHelper determines if any services or fingerprints match the given message
// and runs the processor, returning whether a processor was found
func (h *handler) handleMessageHelper(ecrMsg format.Message, bundle Bundle) bool {
	fingerprint := ecrMsg.GetKeyFP()
	identity := bundle.Identity
	round := bundle.RoundInfo

	jww.INFO.Printf("handleMessage(msgDigest: %s, SIH: %s, KeyFP: %s)",
		ecrMsg.Digest(), fingerprint,
		base64.StdEncoding.EncodeToString(ecrMsg.GetSIH()))

	// If we have a fingerprint, process it
	if proc, exists := h.pop(identity.Source, fingerprint); exists {
		jww.DEBUG.Printf("handleMessage found fingerprint: %s",
			ecrMsg.Digest())
		proc.Process(ecrMsg, nil, nil, identity, round)
		return true
	}

	services, tags, metadata, exists := h.get(
		identity.Source, ecrMsg.GetSIH(), ecrMsg.GetContents())
	// If the id doesn't exist or there are no services for it, then
	// we want messages to be reprocessed as garbled.
	if exists && len(services) != 0 {
		for _, t := range services {
			jww.DEBUG.Printf("handleMessage service found: %s, %s",
				ecrMsg.Digest(), t)
			go t.Process(ecrMsg, tags, metadata, identity, round)
		}
		return true
	}

	// handle the fallthrough, if it exists
	if p, exist := h.getFallthrough(identity.Source); exist {
		p.Process(ecrMsg, nil, nil, identity, round)
		return true
	}

	im := fmt.Sprintf("Message cannot be identified: keyFP: %v, round: %d "+
		"msgDigest: %s, not determined to be for client",
		ecrMsg.GetKeyFP(), bundle.Round, ecrMsg.Digest())
	jww.TRACE.Printf(im)

	h.events.Report(1, "MessageReception", "Garbled", im)

	return false
}
