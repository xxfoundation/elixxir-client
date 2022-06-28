////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"encoding/json"
	"gitlab.com/elixxir/client/xxdk"
	"io"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/catalog"
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
	callback := getClientAuthCallback(cb, nil, e2eClient.GetE2E(), e2eClient.GetAuth(), p)
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

// StartServer assembles a Connection object on the reception-side and feeds it
// into the given Callback whenever an incoming request for an E2E partnership
// with a partner.Manager is confirmed.
//
// It is recommended that this be called before StartNetworkFollower to ensure
// no requests are missed.
// This call does an xxDK.ephemeralLogin under the hood and the connection
// server must be the only listener on auth.
func StartServer(identity xxdk.ReceptionIdentity, cb Callback, net *xxdk.Cmix,
	p Params) (*xxdk.E2e, error) {

	// Build callback for E2E negotiation
	callback := getServerAuthCallback(nil, cb, p)

	// Return an ephemeral E2e object
	return xxdk.LoginEphemeral(net, callback, identity)
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
