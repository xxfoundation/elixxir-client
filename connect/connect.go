////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"io"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
)

const (
	// connectionTimeout is the time.Duration for a connection
	// to be established before the requester times out.
	connectionTimeout = 15 * time.Second
)

// Connection is a wrapper for the E2E and auth packages.
// It can be used to automatically establish an E2E partnership
// with a partner.Manager, or be built from an existing E2E partnership.
// You can then use this interface to send to and receive from the
// newly-established partner.Manager.
type Connection interface {
	// Closer deletes this Connection's partner.Manager and releases resources
	io.Closer

	// GetPartner returns the partner.Manager for this Connection
	GetPartner() partner.Manager

	// SendE2E is a wrapper for sending specifically to the Connection's
	// partner.Manager
	SendE2E(mt catalog.MessageType, payload []byte, params clientE2e.Params) (
		[]id.Round, e2e.MessageID, time.Time, error)

	// RegisterListener is used for E2E reception
	// and allows for reading data sent from the partner.Manager
	RegisterListener(messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
	// Unregister listener for E2E reception
	Unregister(listenerID receive.ListenerID)

	// FirstPartitionSize returns the max partition payload size for the
	// first payload
	FirstPartitionSize() uint

	// SecondPartitionSize returns the max partition payload size for all
	// payloads after the first payload
	SecondPartitionSize() uint

	// PartitionSize returns the partition payload size for the given
	// payload index. The first payload is index 0.
	PartitionSize(payloadIndex uint) uint

	// PayloadSize Returns the max payload size for a partitionable E2E
	// message
	PayloadSize() uint
}

// Callback is the callback format required to retrieve
// new Connection objects as they are established.
type Callback func(connection Connection)

// Params for managing Connection objects.
type Params struct {
	Auth    auth.Params
	Rekey   rekey.Params
	Event   event.Reporter `json:"-"`
	Timeout time.Duration
}

// GetDefaultParams returns a usable set of default Connection parameters.
func GetDefaultParams() Params {
	return Params{
		Auth:    auth.GetDefaultTemporaryParams(),
		Rekey:   rekey.GetDefaultEphemeralParams(),
		Event:   event.NewEventManager(),
		Timeout: connectionTimeout,
	}
}

// GetParameters returns the default Params, or override with given
// parameters, if set.
func GetParameters(params string) (Params, error) {
	p := GetDefaultParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Params{}, err
		}
	}
	return p, nil
}

// Connect performs auth key negotiation with the given recipient,
// and returns a Connection object for the newly-created partner.Manager
// This function is to be used sender-side and will block until the
// partner.Manager is confirmed.
func Connect(recipient contact.Contact, e2eClient *xxdk.E2e,
	p Params) (Connection, error) {

	// Build callback for E2E negotiation
	signalChannel := make(chan Connection, 1)
	cb := func(connection Connection) {
		signalChannel <- connection
	}
	callback := getAuthCallback(cb, nil, e2eClient.GetE2E(), e2eClient.GetAuth(), p)
	e2eClient.GetAuth().AddPartnerCallback(recipient.ID, callback)

	// Perform the auth request
	_, err := e2eClient.GetAuth().Request(recipient, nil)
	if err != nil {
		return nil, err
	}

	// Block waiting for auth to confirm
	jww.DEBUG.Printf("Connection waiting for auth request "+
		"for %s to be confirmed...", recipient.ID.String())
	timeout := time.NewTimer(p.Timeout)
	defer timeout.Stop()
	select {
	case newConnection := <-signalChannel:
		// Verify the Connection is complete
		if newConnection == nil {
			return nil, errors.Errorf("Unable to complete connection "+
				"with partner %s", recipient.ID.String())
		}
		jww.DEBUG.Printf("Connection auth request for %s confirmed",
			recipient.ID.String())
		return newConnection, nil
	case <-timeout.C:
		return nil, errors.Errorf("Connection request with "+
			"partner %s timed out", recipient.ID.String())
	}
}

// StartServer assembles a Connection object on the reception-side
// and feeds it into the given Callback whenever an incoming request
// for an E2E partnership with a partner.Manager is confirmed.
func StartServer(cb Callback, myId *id.ID, privKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p Params) error {

	// Build an ephemeral KV
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Build E2e handler
	err := clientE2e.Init(kv, myId, privKey, grp, p.Rekey)
	if err != nil {
		return err
	}
	e2eHandler, err := clientE2e.Load(kv, net, myId, grp, rng, p.Event)
	if err != nil {
		return err
	}

	// Build callback for E2E negotiation
	callback := getAuthCallback(nil, cb, e2eHandler, nil, p)

	// Build auth object for E2E negotiation
	authState, err := auth.NewState(kv, net, e2eHandler,
		rng, p.Event, p.Auth, callback, nil)
	callback.authState = authState
	return err
}

// handler provides an implementation for the Connection interface.
type handler struct {
	auth    auth.State
	partner partner.Manager
	e2e     clientE2e.Handler
	params  Params
}

// BuildConnection assembles a Connection object
// after an E2E partnership has already been confirmed with the given
// partner.Manager.
func BuildConnection(partner partner.Manager, e2eHandler clientE2e.Handler,
	auth auth.State, p Params) Connection {
	return &handler{
		auth:    auth,
		partner: partner,
		params:  p,
		e2e:     e2eHandler,
	}
}

// Close deletes this Connection's partner.Manager and releases resources.
func (h *handler) Close() error {
	if err := h.e2e.DeletePartner(h.partner.PartnerId()); err != nil {
		return err
	}
	return h.auth.Close()
}

// GetPartner returns the partner.Manager for this Connection.
func (h *handler) GetPartner() partner.Manager {
	return h.partner
}

// SendE2E is a wrapper for sending specifically to the Connection's
// partner.Manager.
func (h *handler) SendE2E(mt catalog.MessageType, payload []byte,
	params clientE2e.Params) (
	[]id.Round, e2e.MessageID, time.Time, error) {
	return h.e2e.SendE2E(mt, h.partner.PartnerId(), payload, params)
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager.
func (h *handler) RegisterListener(messageType catalog.MessageType,
	newListener receive.Listener) receive.ListenerID {
	return h.e2e.RegisterListener(h.partner.PartnerId(),
		messageType, newListener)
}

// Unregister listener for E2E reception.
func (h *handler) Unregister(listenerID receive.ListenerID) {
	h.e2e.Unregister(listenerID)
}

// authCallback provides callback functionality for interfacing between
// auth.State and Connection. This is used both for blocking creation of a
// Connection object until the auth Request is confirmed and for dynamically
// building new Connection objects when an auth Request is received.
type authCallback struct {
	// Used for signaling confirmation of E2E partnership
	confirmCallback Callback
	requestCallback Callback

	// Used for building new Connection objects
	connectionE2e    clientE2e.Handler
	connectionParams Params
	authState        auth.State
}

// getAuthCallback returns a callback interface to be passed into the creation
// of an auth.State object.
// it will accept requests only if a request callback is passed in
func getAuthCallback(confirm, request Callback, e2e clientE2e.Handler,
	auth auth.State, params Params) *authCallback {
	return &authCallback{
		confirmCallback:  confirm,
		requestCallback:  request,
		connectionE2e:    e2e,
		connectionParams: params,
		authState:        auth,
	}
}

// Confirm will be called when an auth Confirm message is processed.
func (a authCallback) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	jww.DEBUG.Printf("Connection auth request for %s confirmed",
		requestor.ID.String())
	defer a.authState.DeletePartnerCallback(requestor.ID)

	// After confirmation, get the new partner
	newPartner, err := a.connectionE2e.GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		if a.confirmCallback != nil {
			a.confirmCallback(nil)
		}
		return
	}

	// Return the new Connection object
	if a.confirmCallback != nil {
		a.confirmCallback(BuildConnection(newPartner, a.connectionE2e,
			a.authState, a.connectionParams))
	}
}

// Request will be called when an auth Request message is processed.
func (a authCallback) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	if a.requestCallback == nil {
		jww.ERROR.Printf("Received a request when requests are" +
			"not enable, will not accept")
	}
	_, err := a.authState.Confirm(requestor)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.requestCallback(nil)
	}
	// After confirmation, get the new partner
	newPartner, err := a.connectionE2e.GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.requestCallback(nil)

		return
	}

	// Return the new Connection object
	a.requestCallback(BuildConnection(newPartner, a.connectionE2e,
		a.authState, a.connectionParams))
}

// Reset will be called when an auth Reset operation occurs.
func (a authCallback) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// FirstPartitionSize returns the max partition payload size for the
// first payload
func (h *handler) FirstPartitionSize() uint {
	return h.e2e.FirstPartitionSize()
}

// SecondPartitionSize returns the max partition payload size for all
// payloads after the first payload
func (h *handler) SecondPartitionSize() uint {
	return h.e2e.SecondPartitionSize()
}

// PartitionSize returns the partition payload size for the given
// payload index. The first payload is index 0.
func (h *handler) PartitionSize(payloadIndex uint) uint {
	return h.e2e.PartitionSize(payloadIndex)
}

// PayloadSize Returns the max payload size for a partition-able E2E
// message
func (h *handler) PayloadSize() uint {
	return h.e2e.PayloadSize()
}
