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
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/connections/connect"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
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
// of establishing a connection. This will establish a connect.Connection with
// the server and then authenticate their identity to the server.
func ConnectWithAuthentication(recipient contact.Contact, myId *id.ID,
	salt []byte, rsaPrivkey *rsa.PrivateKey, dhPrivKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) (Connection, error) {

	conn, err := connect.Connect(recipient, myId, dhPrivKey, rng, grp, net, p)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to establish connection "+
			"with recipient %s", recipient.ID)
	}
	// Build the callback for the connection being established
	authConnChan := make(chan Connection, 1)
	cb := ConnectionCallback(func(authConn Connection) {
		authConnChan <- authConn
	})

	go initiateClientAuthentication(cb, conn, net, rng, rsaPrivkey, salt, p)

	// Block waiting for auth to confirm it timeouts
	jww.DEBUG.Printf("Connection waiting for authenticated "+
		"connection with %s to be established...", recipient.ID.String())
	timeout := time.NewTimer(p.Timeout)
	defer timeout.Stop()
	select {
	case authConn := <-authConnChan:
		if authConn == nil {
			return nil, errors.Errorf(
				"Unable to complete authenticated connection with partner %s",
				recipient.ID.String())
		}
		return authConn, nil
	case <-timeout.C:
		return nil, errors.Errorf("Authenticated connection with "+
			"partner %s timed out", recipient.ID.String())
	}

}

// StartAuthenticatedConnectionServer is called by the receiver of an
// authenticated connection request. Calling this will indicate that they
// will handle authenticated requests and verify the client's attempt to
// authenticate themselves. An established authenticated.Connection will
// be passed via the callback.
func StartAuthenticatedConnectionServer(cb ConnectionCallback,
	myId *id.ID, privKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) error {

	// Register the waiter for a connection establishment
	connChan := make(chan connect.Connection, 1)
	connCb := connect.Callback(func(connection connect.Connection) {
		connChan <- connection
	})
	err := connect.RegisterConnectionCallback(connCb, myId, privKey, rng, grp, net, p)
	if err != nil {
		return err
	}

	// Wait for a connection to be established
	timer := time.NewTimer(p.Timeout)
	defer timer.Stop()
	select {
	case conn := <-connChan:
		// Upon establishing a connection, register a listener for the
		// client's identity proof. If a identity authentication
		// message is received and validated, an authenticated connection will be
		// passed along via the callback
		conn.RegisterListener(catalog.ConnectionAuthenticationRequest,
			getServer(cb, conn))
		return nil
	case <-timer.C:
		return errors.New("Timed out trying to establish a connection")
	}
}

// handler provides an implementation for the authenticated.Connection
// interface.
type handler struct {
	connect.Connection
	isAuthenticated bool
	authMux         sync.Mutex
}

// buildAuthenticatedConnection assembles an authenticated.Connection object.
func buildAuthenticatedConnection(conn connect.Connection) *handler {
	return &handler{
		Connection:      conn,
		isAuthenticated: false,
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
