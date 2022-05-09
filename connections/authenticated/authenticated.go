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
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
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

// Callback is the callback format required to retrieve
// new authenticated.Connection objects as they are established.
type Callback func(connection Connection)

// ConnectWithAuthentication is called by the client, ie the one establishing
// connection with the server. Once a connect.Connection has been established
// with the server and then authenticate their identity to the server.
func ConnectWithAuthentication(recipient contact.Contact, myId *id.ID,
	salt []byte, myRsaPrivKey *rsa.PrivateKey, myDhPrivKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) (Connection, error) {

	conn, err := connect.Connect(recipient, myId, myDhPrivKey, rng, grp, net, p)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to establish connection "+
			"with recipient %s", recipient.ID)
	}

	// Construct message
	payload, err := makeClientAuthRequest(conn.GetPartner(), rng, myRsaPrivKey, salt)
	if err != nil {
		errClose := conn.Close()
		if errClose != nil {
			return nil, errors.Errorf(
				"failed to close connection with %s after error %v: %+v",
				recipient.ID, err, errClose)
		}
		return nil, errors.WithMessagef(err, "failed to construct client "+
			"authentication message")
	}

	// Send message to user
	e2eParams := clientE2e.GetDefaultParams()
	rids, _, _, err := conn.SendE2E(catalog.ConnectionAuthenticationRequest,
		payload, e2eParams)
	if err != nil {
		errClose := conn.Close()
		if errClose != nil {
			return nil, errors.Errorf(
				"failed to close connection with %s after error %v: %+v",
				recipient.ID, err, errClose)
		}
		return nil, errors.WithMessagef(err, "failed to construct client "+
			"authentication message")
	}

	// Record since we first successfully sen the message
	timeStart := netTime.Now()

	// Determine that the message is properly sent by tracking the success
	// of the round(s)
	authConnChan := make(chan Connection, 1)
	roundCb := cmix.RoundEventCallback(func(allRoundsSucceeded,
		timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		if allRoundsSucceeded {
			// If rounds succeeded, assume recipient has successfully
			// confirmed the authentication. Pass the connection
			// along via the callback
			authConn := buildAuthenticatedConnection(conn)
			authConn.setAuthenticated()
			authConnChan <- authConn
		}
	})

	// Find the remaining time in the timeout since we first sent the message
	remainingTime := e2eParams.Timeout - netTime.Since(timeStart)

	// Track the result of the round(s) we sent the
	// identity authentication message on
	err = net.GetRoundResults(remainingTime,
		roundCb, rids...)
	if err != nil {
		return nil, errors.Errorf("could not track rounds for successful " +
			"identity confirmation message delivery")
	}
	// Block waiting for confirmation of the round(s) success (or timeout
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

// StartServer is called by the receiver of an
// authenticated connection request. Calling this will indicate that they
// will handle authenticated requests and verify the client's attempt to
// authenticate themselves. An established authenticated.Connection will
// be passed via the callback.
func StartServer(cb Callback,
	myId *id.ID, privKey *cyclic.Int,
	rng *fastRNG.StreamGenerator, grp *cyclic.Group, net cmix.Client,
	p connect.Params) error {

	// Register the waiter for a connection establishment
	connCb := connect.Callback(func(connection connect.Connection) {
		// Upon establishing a connection, register a listener for the
		// client's identity proof. If a identity authentication
		// message is received and validated, an authenticated connection will
		// be passed along via the callback
		connection.RegisterListener(catalog.ConnectionAuthenticationRequest,
			buildAuthConfirmationHandler(cb, connection))
	})
	return connect.StartServer(connCb, myId, privKey, rng, grp,
		net, p)
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
