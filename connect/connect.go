////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"io"
	"sync/atomic"
	"time"

	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/xx_network/primitives/netTime"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/catalog"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
)

var alreadyClosedErr = errors.New("connection is closed")

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
		newListener receive.Listener) (receive.ListenerID, error)
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

	// LastUse returns the timestamp of the last time the connection was
	// utilised.
	LastUse() time.Time
}

// Callback is the callback format required to retrieve
// new Connection objects as they are established.
type Callback func(connection Connection)

// Connect performs auth key negotiation with the given recipient,
// and returns a Connection object for the newly-created partner.Manager
// This function is to be used sender-side and will block until the
// partner.Manager is confirmed.
func Connect(recipient contact.Contact, user *xxdk.E2e,
	p xxdk.E2EParams) (Connection, error) {
	// Build callback for E2E negotiation
	signalChannel := make(chan Connection, 1)
	cb := func(connection Connection) {
		signalChannel <- connection
	}
	callback := getClientAuthCallback(cb, user.GetE2E(),
		user.GetAuth(), p)
	user.GetAuth().AddPartnerCallback(recipient.ID, callback)

	// Perform the auth request
	_, err := user.GetAuth().Reset(recipient)
	if err != nil {
		return nil, err
	}

	// Block waiting for auth to confirm
	jww.DEBUG.Printf("Connection waiting for auth request "+
		"for %s to be confirmed...", recipient.ID.String())
	timeout := time.NewTimer(p.Base.Timeout)
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
//
// This calls xxdk.LoginEphemeral under the hood and the connection
// server must be the only listener on auth.
func StartServer(identity xxdk.ReceptionIdentity, connectionCallback Callback,
	net *xxdk.Cmix, params xxdk.E2EParams, clParams ConnectionListParams) (*ConnectionServer, error) {

	// Create connection list and start cleanup thread
	cl := NewConnectionList(clParams)
	err := net.AddService(cl.CleanupThread)
	if err != nil {
		return nil, err
	}

	// Build callback for E2E negotiation
	callback := getServerAuthCallback(connectionCallback, cl, params)

	e2eClient, err := xxdk.LoginEphemeral(net, callback, identity, params)
	if err != nil {
		return nil, err
	}

	// Return an ephemeral E2e object
	return &ConnectionServer{e2eClient, cl}, nil
}

// ConnectionServer contains
type ConnectionServer struct {
	User *xxdk.E2e
	Cl   *ConnectionList
}

// handler provides an implementation for the Connection interface.
type handler struct {
	auth    auth.State
	partner partner.Manager
	e2e     clientE2e.Handler
	params  xxdk.E2EParams

	// Timestamp of last time a message was sent or received (Unix nanoseconds)
	lastUse *int64

	// Indicates if the connection has been closed (0 = open, 1 = closed)
	closed *uint32
}

// BuildConnection assembles a Connection object
// after an E2E partnership has already been confirmed with the given
// partner.Manager.
func BuildConnection(partner partner.Manager, e2eHandler clientE2e.Handler,
	auth auth.State, p xxdk.E2EParams) Connection {
	lastUse := netTime.Now().UnixNano()
	closed := uint32(0)
	return &handler{
		auth:    auth,
		partner: partner,
		params:  p,
		e2e:     e2eHandler,
		lastUse: &lastUse,
		closed:  &closed,
	}
}

// Close deletes this Connection's partner.Manager and releases resources. If
// the connection is already closed, then nil is returned.
func (h *handler) Close() error {
	if h.isClosed() {
		return nil
	}

	// Get partner ID once at the top because PartnerId makes a copy
	partnerID := h.partner.PartnerId()

	// Unregister all listeners
	h.e2e.UnregisterUserListeners(partnerID)

	// Delete partner from e2e and auth
	if err := h.e2e.DeletePartner(partnerID); err != nil {
		return err
	}
	if err := h.auth.DeletePartner(partnerID); err != nil {
		return err
	}

	atomic.StoreUint32(h.closed, 1)

	return nil
}

// GetPartner returns the partner.Manager for this Connection.
func (h *handler) GetPartner() partner.Manager {
	return h.partner
}

// SendE2E is a wrapper for sending specifically to the Connection's
// partner.Manager.
func (h *handler) SendE2E(mt catalog.MessageType, payload []byte,
	params clientE2e.Params) ([]id.Round, e2e.MessageID, time.Time, error) {
	if h.isClosed() {
		return nil, e2e.MessageID{}, time.Time{}, alreadyClosedErr
	}

	h.updateLastUse(netTime.Now())

	return h.e2e.SendE2E(mt, h.partner.PartnerId(), payload, params)
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager.
func (h *handler) RegisterListener(messageType catalog.MessageType,
	newListener receive.Listener) (receive.ListenerID, error) {
	if h.isClosed() {
		return receive.ListenerID{}, alreadyClosedErr
	}
	lt := &listenerTracker{h, newListener}
	return h.e2e.RegisterListener(h.partner.PartnerId(), messageType, lt), nil
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

// LastUse returns the timestamp of the last time the connection was utilised.
func (h *handler) LastUse() time.Time {
	return time.Unix(0, atomic.LoadInt64(h.lastUse))
}

// updateLastUse updates the last use time stamp to the given time.
func (h *handler) updateLastUse(t time.Time) {
	atomic.StoreInt64(h.lastUse, t.UnixNano())
}

// isClosed returns true if the connection is closed.
func (h *handler) isClosed() bool {
	return atomic.LoadUint32(h.closed) == 1
}
