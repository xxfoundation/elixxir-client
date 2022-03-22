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
	"gitlab.com/elixxir/client/network/nodes"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/rateLimiting"
	"strconv"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
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
	sender           *gateway.Sender
	blacklistedNodes map[string]interface{}

	messageReception chan Bundle
	nodeRegistration chan network.NodeGateway
	networkIsHealthy chan bool
	checkInProgress  chan struct{}

	inProcess *MeteredCmixMessageBuffer

	rng      *fastRNG.StreamGenerator
	events   interfaces.EventManager
	session  *storage.Session
	nodes    nodes.Registrar
	instance *network.Instance

	FingerprintsManager
	TriggersManager

	// sending rate limit tracker
	rateLimitBucket *rateLimiting.Bucket
	rateLimitParams utility.BucketParamStore
}

func NewPickup(param params.Network,
	nodeRegistration chan network.NodeGateway, sender *gateway.Sender,
	session *storage.Session, rng *fastRNG.StreamGenerator,
	events interfaces.EventManager, nodes nodes.Registrar,
	instance *network.Instance) Pickup {

	garbled, err := NewOrLoadMeteredCmixMessageBuffer(session.GetKV(), inProcessKey)
	if err != nil {
		jww.FATAL.Panicf("Failed to load or new the Garbled Messages system: %v", err)
	}

	m := pickup{
		param:            param,
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		networkIsHealthy: make(chan bool, 1),
		checkInProgress:  make(chan struct{}, 100),
		nodeRegistration: nodeRegistration,
		sender:           sender,
		inProcess:        garbled,
		rng:              rng,
		events:           events,
		session:          session,
		nodes:            nodes,
		instance:         instance,
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

// GetMessageReceptionChannel gets the channel to send received messages on.
func (p *pickup) GetMessageReceptionChannel() chan<- Bundle {
	return p.messageReception
}

// StartProcesses starts all worker pool.
func (p *pickup) StartProcesses() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	// create the message handler workers
	for i := uint(0); i < p.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(
			"MessageReception Worker " + strconv.Itoa(int(i)))
		go p.handleMessages(stop)
		multi.Add(stop)
	}

	// create the in progress messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go p.recheckInProgressRunner(garbledStop)
	multi.Add(garbledStop)

	return multi
}
