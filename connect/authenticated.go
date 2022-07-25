///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package connect

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Constant error messages
const (
	roundTrackingTimeoutErr    = "timed out waiting for round results"
	notAllRoundsSucceededErr   = "not all rounds succeeded"
	failedToCloseConnectionErr = "failed to close connection with %s " +
		"after error %v: %+v"
)

// AuthenticatedConnection is a connect.Connection interface that
// has the receiver authenticating their identity back to the
// initiator.
type AuthenticatedConnection interface {
	// Connection is the base Connect API. This allows
	// sending and listening to the partner
	Connection

	// IsAuthenticated is a function which returns whether the
	// authenticated connection has been completely established.
	IsAuthenticated() bool
}

// AuthenticatedCallback is the callback format required to retrieve
// new AuthenticatedConnection objects as they are established.
type AuthenticatedCallback func(connection AuthenticatedConnection)

// ConnectWithAuthentication is called by the client, ie the one establishing
// connection with the server. Once a connect.Connection has been established
// with the server and then authenticate their identity to the server.
func ConnectWithAuthentication(recipient contact.Contact, user *xxdk.E2e,
	p xxdk.E2EParams) (AuthenticatedConnection, error) {

	// Track the time since we started to attempt to establish a connection
	timeStart := netTime.Now()

	// Establish a connection with the server
	conn, err := Connect(recipient, user, p)
	if err != nil {
		return nil, errors.Errorf("failed to establish connection "+
			"with recipient %s: %+v", recipient.ID, err)
	}

	// Build the authenticated connection and return
	identity := user.GetReceptionIdentity()
	privKey, err := identity.GetRSAPrivatePem()
	if err != nil {
		return nil, err
	}
	return connectWithAuthentication(conn, timeStart, recipient,
		identity.Salt, privKey, user.GetRng(), user.GetCmix(), p)
}

// connectWithAuthentication builds and sends an IdentityAuthentication to
// the server. This will wait until the round it sends on completes or a
// timeout occurs.
func connectWithAuthentication(conn Connection, timeStart time.Time,
	recipient contact.Contact, salt []byte, myRsaPrivKey *rsa.PrivateKey,
	rng *fastRNG.StreamGenerator,
	net cmix.Client, p xxdk.E2EParams) (AuthenticatedConnection, error) {
	// Construct message to prove your identity to the server
	payload, err := buildClientAuthRequest(conn.GetPartner(), rng,
		myRsaPrivKey, salt)
	if err != nil {
		// Close connection on an error
		errClose := conn.Close()
		if errClose != nil {
			return nil, errors.Errorf(
				failedToCloseConnectionErr,
				recipient.ID, err, errClose)
		}
		return nil, errors.WithMessagef(err, "failed to construct client "+
			"authentication message")
	}

	// Send message to server
	rids, _, _, err := conn.SendE2E(catalog.ConnectionAuthenticationRequest,
		payload, clientE2e.GetDefaultParams())
	if err != nil {
		// Close connection on an error
		errClose := conn.Close()
		if errClose != nil {
			return nil, errors.Errorf(
				failedToCloseConnectionErr,
				recipient.ID, err, errClose)
		}
		return nil, errors.WithMessagef(err, "failed to send client "+
			"authentication message")
	}

	// Determine that the message is properly sent by tracking the success
	// of the round(s)
	roundErr := make(chan error, 1)
	roundCb := cmix.RoundEventCallback(func(allRoundsSucceeded,
		timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		// Check for failures while tracking rounds
		if timedOut || !allRoundsSucceeded {
			if timedOut {
				roundErr <- errors.New(roundTrackingTimeoutErr)
			} else {
				// If we did not time out, then not all rounds succeeded
				roundErr <- errors.New(notAllRoundsSucceededErr)
			}
			return
		}

		// If no errors occurred, signal so; an authenticated channel may
		// be constructed now
		roundErr <- nil
	})

	// Find the remaining time in the timeout since we first sent the message
	remainingTime := p.Base.Timeout - netTime.Since(timeStart)

	// Track the result of the round(s) we sent the
	// identity authentication message on
	err = net.GetRoundResults(remainingTime,
		roundCb, rids...)
	if err != nil {
		return nil, errors.Errorf("could not track rounds for successful " +
			"identity confirmation message delivery")
	}
	// Block waiting for confirmation of the round(s) success (or timeout
	jww.DEBUG.Printf("AuthenticatedConnection waiting for authenticated "+
		"connection with %s to be established...", recipient.ID.String())
	// Wait for the round callback to send a round error
	err = <-roundErr
	if err != nil {
		// Close connection on an error
		errClose := conn.Close()
		if errClose != nil {
			return nil, errors.Errorf(
				failedToCloseConnectionErr,
				recipient.ID, err, errClose)
		}

		return nil, errors.Errorf("failed to confirm if identity "+
			"authentication message was sent to %s: %v", recipient.ID, err)
	}

	// If channel received no error, construct and return the
	// authenticated connection
	authConn := buildAuthenticatedConnection(conn)
	authConn.setAuthenticated()
	return authConn, nil
}

// StartAuthenticatedServer is called by the receiver of an
// authenticated connection request. Calling this will indicate that they
// will handle authenticated requests and verify the client's attempt to
// authenticate themselves. An established AuthenticatedConnection will
// be passed via the callback.
func StartAuthenticatedServer(identity xxdk.ReceptionIdentity,
	authCb AuthenticatedCallback, net *xxdk.Cmix, p xxdk.E2EParams,
	clParams ConnectionListParams) (
	*ConnectionServer, error) {

	// Register the waiter for a connection establishment
	connCb := Callback(func(connection Connection) {
		// Upon establishing a connection, register a listener for the
		// client's identity proof. If an identity authentication
		// message is received and validated, an authenticated connection will
		// be passed along via the AuthenticatedCallback
		_, err := connection.RegisterListener(
			catalog.ConnectionAuthenticationRequest,
			buildAuthConfirmationHandler(authCb, connection))
		if err != nil {
			jww.ERROR.Printf(
				"Failed to register listener on connection with %s: %+v",
				connection.GetPartner().PartnerId(), err)
		}
	})
	return StartServer(identity, connCb, net, p, clParams)
}

// authenticatedHandler provides an implementation for the
// AuthenticatedConnection interface.
type authenticatedHandler struct {
	Connection
	isAuthenticated bool
	authMux         sync.Mutex
}

// buildAuthenticatedConnection assembles an AuthenticatedConnection object.
func buildAuthenticatedConnection(conn Connection) *authenticatedHandler {
	return &authenticatedHandler{
		Connection:      conn,
		isAuthenticated: false,
	}
}

// IsAuthenticated returns whether the AuthenticatedConnection has completed
// the authentication process.
func (h *authenticatedHandler) IsAuthenticated() bool {
	return h.isAuthenticated
}

// setAuthenticated is a helper function which sets the
// AuthenticatedConnection as authenticated.
func (h *authenticatedHandler) setAuthenticated() {
	h.authMux.Lock()
	defer h.authMux.Unlock()
	h.isAuthenticated = true
}
