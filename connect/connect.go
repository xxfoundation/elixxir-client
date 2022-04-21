////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"time"
)

// Connection is a wrapper for the E2E and auth packages.
// It can be used to automatically establish an E2E partnership
// with a partner.Manager, or be built from an existing E2E partnership.
// You can then use this interface to send to and receive from the newly-established partner.Manager.
type Connection interface {
	// Closer deletes this Connection's partner.Manager and releases resources
	io.Closer

	// GetPartner returns the partner.Manager for this Connection
	GetPartner() partner.Manager

	// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
	SendE2E(mt catalog.MessageType, payload []byte, params clientE2e.Params) (
		[]id.Round, e2e.MessageID, time.Time, error)

	// RegisterListener is used for E2E reception
	// and allows for reading data sent from the partner.Manager
	RegisterListener(messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
	// Unregister listener for E2E reception
	Unregister(listenerID receive.ListenerID)
}

// handler provides an implementation for the Connection interface
type handler struct {
	partner partner.Manager
	e2e     clientE2e.Handler
	params  Params
}

// Params for managing Connection objects
type Params struct {
	auth  auth.Param
	event event.Reporter
}

// GetDefaultParams returns a usable set of default Connection parameters
func GetDefaultParams() Params {
	return Params{
		auth:  auth.GetDefaultParams(),
		event: nil,
	}
}

// Connect performs auth key negotiation with the given recipient,
// and returns a Connection object for the newly-created partner.Manager
// This function is to be used sender-side and will block until the partner.Manager is confirmed
func Connect(recipient contact.Contact, myId *id.ID, rng *fastRNG.StreamGenerator,
	grp *cyclic.Group, net cmix.Client, p Params) (Connection, error) {

	// Build an ephemeral KV
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Build E2e handler
	e2eHandler, err := clientE2e.Load(kv, net, myId, grp, rng, p.event)
	if err != nil {
		return nil, err
	}

	// Build callback for E2E negotiation
	callback := getAuthCallback()

	// Build auth object for E2E negotiation
	authState, err := auth.NewState(kv, net, e2eHandler,
		rng, p.event, p.auth, callback, nil)
	if err != nil {
		return nil, err
	}

	// Perform the auth request
	_, err = authState.Request(recipient, nil)
	if err != nil {
		return nil, err
	}

	// Block waiting for auth to confirm
	jww.DEBUG.Printf("Connection waiting for auth request for %s to be confirmed...", recipient.ID.String())
	<-callback.confirmPartner
	jww.DEBUG.Printf("Connection auth request for %s confirmed", recipient.ID.String())

	// After confirmation, get the new partner
	newPartner, err := e2eHandler.GetPartner(recipient.ID)
	if err != nil {
		return nil, err
	}

	return BuildConnection(newPartner, e2eHandler, p), nil
}

// WaitForConnections assembles a Connection object on the reception-side
// when an incoming request for an E2E partnership with a partner.Manager is confirmed.
func WaitForConnections(myId *id.ID, rng *fastRNG.StreamGenerator,
	grp *cyclic.Group, net cmix.Client, p Params) (Connection, error) {

	// Build an ephemeral KV
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Build E2e handler
	e2eHandler, err := clientE2e.Load(kv, net, myId, grp, rng, p.event)
	if err != nil {
		return nil, err
	}

	// Build callback for E2E negotiation
	callback := getAuthCallback()

	// Build auth object for E2E negotiation
	_, err = auth.NewState(kv, net, e2eHandler,
		rng, p.event, p.auth, callback, nil)
	if err != nil {
		return nil, err
	}

	// Block waiting for incoming auth request
	jww.DEBUG.Printf("Connection waiting for auth request to be received...")
	newPartnerId := <-callback.confirmPartner
	jww.DEBUG.Printf("Connection auth request for %s confirmed", newPartnerId.String())

	// After confirmation, get the new partner
	newPartner, err := e2eHandler.GetPartner(newPartnerId)
	if err != nil {
		return nil, err
	}

	// Return the new Connection object
	return BuildConnection(newPartner, e2eHandler, p), nil
}

// BuildConnection assembles a Connection object
// after an E2E partnership has already been confirmed with the given partner.Manager
func BuildConnection(partner partner.Manager, e2eHandler clientE2e.Handler, p Params) Connection {
	return &handler{
		partner: partner,
		params:  p,
		e2e:     e2eHandler,
	}
}

// Close deletes this Connection's partner.Manager and releases resources
func (h *handler) Close() error {
	return h.e2e.DeletePartner(h.partner.PartnerId())
}

// GetPartner returns the partner.Manager for this Connection
func (h *handler) GetPartner() partner.Manager {
	return h.partner
}

// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
func (h *handler) SendE2E(mt catalog.MessageType, payload []byte, params clientE2e.Params) (
	[]id.Round, e2e.MessageID, time.Time, error) {
	return h.e2e.SendE2E(mt, h.partner.PartnerId(), payload, params)
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
func (h *handler) RegisterListener(messageType catalog.MessageType, newListener receive.Listener) receive.ListenerID {
	return h.e2e.RegisterListener(h.partner.PartnerId(), messageType, newListener)
}

// Unregister listener for E2E reception
func (h *handler) Unregister(listenerID receive.ListenerID) {
	h.e2e.Unregister(listenerID)
}

// authCallback provides callback functionality for interfacing between auth.State and Connection
// This is used both for blocking creation of a Connection object until the auth Request is confirmed
// and for dynamically building new Connection objects when an auth Request is received.
type authCallback struct {
	// Used for signaling confirmation of E2E partnership
	confirmPartner chan *id.ID
}

// getAuthCallback returns a callback interface to be passed into the creation of an auth.State object.
func getAuthCallback() authCallback {
	return authCallback{
		confirmPartner: make(chan *id.ID, 10),
	}
}

// Confirm will be called when an auth Confirm message is processed
func (c authCallback) Confirm(requestor contact.Contact, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	// Signal to a listening thread that the partnership is confirmed
	c.confirmPartner <- requestor.ID
}

// Request will be called when an auth Request message is processed
func (c authCallback) Request(requestor contact.Contact, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// Reset will be called when an auth Reset operation occurs
func (c authCallback) Reset(requestor contact.Contact, receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}
