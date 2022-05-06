///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package authenticated

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/connections/connect"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// Connection is a connect.Connection interface that
// has the receiver authenticating their identity back to the
// initiator.
type Connection interface {
	// Connection is the base connections API. This allows
	// sending and listening to the partner
	connect.Connection

	// IsAuthenticated is a function which returns whether the
	// authenticated connection has been completely established.
	IsAuthenticated() bool
}

// ConnectionCallback is the callback format required to retrieve
// new authenticated.Connection objects as they are established.
type ConnectionCallback func(connection Connection)

// ConnectWithAuthentication is called by the client, ie the initiator
// of an authenticated.Connection.  This will send a request for an
// authenticated connection to the server.
func ConnectWithAuthentication(recipient contact.Contact,
	myId *id.ID, dhPrivKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) (Connection, error) {

	// Build an ephemeral KV
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Build E2e handler
	err := clientE2e.Init(kv, myId, dhPrivKey, grp, p.Rekey)
	if err != nil {
		return nil, err
	}
	e2eHandler, err := clientE2e.Load(kv, net, myId, grp, rng, p.Event)
	if err != nil {
		return nil, err
	}

	// Build the callback for the connection being established
	authConnChan := make(chan Connection, 1)
	cb := func(authConn Connection) {
		authConnChan <- authConn
	}

	// Build callback for E2E negotiation in the auth package
	clientHandler := getClient(cb, e2eHandler, p)

	// Register a listener for the server's response
	e2eHandler.RegisterListener(recipient.ID,
		catalog.ConnectionAuthenticationRequest,
		clientHandler)

	// Build auth object for E2E negotiation
	authState, err := auth.NewState(kv, net, e2eHandler,
		rng, p.Event, p.Auth, clientHandler, nil)
	if err != nil {
		return nil, err
	}

	// Perform the auth request
	_, err = authState.Request(recipient, nil)
	if err != nil {
		return nil, err
	}

	// Block waiting for auth to confirm
	jww.DEBUG.Printf("Connection waiting for authenticated "+
		"connection with %s to be established...", recipient.ID.String())

	timeout := time.NewTimer(p.Timeout)
	select {
	case newConnection := <-authConnChan:
		if newConnection == nil {
			return nil, errors.Errorf(
				"Unable to complete authenticated connection with partner %s",
				recipient.ID.String())
		}

		return newConnection, nil
	case <-timeout.C:
		return nil, errors.Errorf("Authenticated connection with "+
			"partner %s timed out", recipient.ID.String())
	}
}

// StartAuthenticatedConnectionServer is Called by the receiver of an
// authenticated connection request. Calling this indicated that they will
// recognize and respond to identity authentication requests by
// a client.
func StartAuthenticatedConnectionServer(cb ConnectionCallback,
	myId *id.ID, salt []byte, rsaPrivkey *rsa.PrivateKey, privKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) error {

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
	callback := getServer(cb, e2eHandler, net, rsaPrivkey, rng, p)

	// Build auth object for E2E negotiation
	_, err = auth.NewState(kv, net, e2eHandler,
		rng, p.Event, p.Auth, callback, nil)

	return err
}

// handler provides an implementation for the authenticated.Connection
// interface.
type handler struct {
	connect.Connection
	isAuthenticated bool
	authMux         sync.Mutex
}

// buildAuthenticatedConnection assembles an authenticated.Connection object.
// This is called by the connection server once it has sent an
// IdentityAuthentication to the client.
// This is called by the client when they have received and confirmed the data within
// a IdentityAuthentication message.
func buildAuthenticatedConnection(partner partner.Manager,
	e2eHandler clientE2e.Handler,
	p connect.Params) *handler {

	return &handler{
		Connection: connect.BuildConnection(partner, e2eHandler, p),
	}
}

// IsAuthenticated returns whether the Connection has completed the authentication
// process.
func (h *handler) IsAuthenticated() bool {
	return h.isAuthenticated
}

// setAuthenticated is a helper function which sets the Connection as authenticated.
func (h *handler) setAuthenticated() {
	h.authMux.Lock()
	defer h.authMux.Unlock()
	h.isAuthenticated = true
}
