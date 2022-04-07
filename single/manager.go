///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

const (
	rawMessageBuffSize           = 100
	singleUseTransmission        = "SingleUseTransmission"
	singleUseReceiveTransmission = "SingleUseReceiveTransmission"
	singleUseResponse            = "SingleUseResponse"
	singleUseReceiveResponse     = "SingleUseReceiveResponse"
	singleUseStop                = "SingleUse"
)

// Manager handles the transmission and reception of single-use communication.
type Manager struct {
	// Client and its field
	client    *api.Client
	store     *storage.Session
	reception *receptionID.Store
	swb       interfaces.Switchboard
	net       interfaces.NetworkManager
	rng       *fastRNG.StreamGenerator

	// Holds the information needed to manage each pending communication. A
	// state is created when a transmission is started and is removed on
	// response or timeout.
	p *pending

	// List of callbacks that can be called when a transmission is received. For
	// an entity to receive a message, it must register a callback in this map
	// with the same tag used to send the message.
	callbackMap *callbackMap
}

// NewManager creates a new single-use communication manager.
func NewManager(client *api.Client) *Manager {
	return newManager(client, client.GetStorage().Reception())
}

func newManager(client *api.Client, reception *receptionID.Store) *Manager {
	return &Manager{
		client:      client,
		store:       client.GetStorage(),
		reception:   reception,
		swb:         client.GetSwitchboard(),
		net:         client.GetNetworkInterface(),
		rng:         client.GetRng(),
		p:           newPending(),
		callbackMap: newCallbackMap(),
	}
}

// StartProcesses starts the process of receiving single-use transmissions and
// replies.
func (m *Manager) StartProcesses() (stoppable.Stoppable, error) {
	// Start waiting for single-use transmission
	transmissionStop := stoppable.NewSingle(singleUseTransmission)
	transmissionChan := make(chan message.Receive, rawMessageBuffSize)
	m.swb.RegisterChannel(singleUseReceiveTransmission, &id.ID{}, message.Raw, transmissionChan)
	go m.receiveTransmissionHandler(transmissionChan, transmissionStop)

	// Start waiting for single-use response
	responseStop := stoppable.NewSingle(singleUseResponse)
	responseChan := make(chan message.Receive, rawMessageBuffSize)
	m.swb.RegisterChannel(singleUseReceiveResponse, &id.ID{}, message.Raw, responseChan)
	go m.receiveResponseHandler(responseChan, responseStop)

	// Create a multi stoppable
	singleUseMulti := stoppable.NewMulti(singleUseStop)
	singleUseMulti.Add(transmissionStop)
	singleUseMulti.Add(responseStop)

	return singleUseMulti, nil
}

// RegisterCallback registers a callback for received messages.
func (m *Manager) RegisterCallback(tag string, callback ReceiveComm) {
	jww.DEBUG.Printf("Registering single-use callback with tag %s.", tag)
	m.callbackMap.registerCallback(tag, callback)
}
