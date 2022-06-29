///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/receive"
	connCrypto "gitlab.com/elixxir/crypto/connect"
	"gitlab.com/xx_network/primitives/id"
)

// authenticatedServerListenerName is the name of the client's
//listener interface.
const authenticatedServerListenerName = "AuthenticatedServerListener"

// server is an interface that wraps receive.Listener. This handles
// the server listening for the client's proof of identity message.
type server interface {
	receive.Listener
}

// serverListener provides an implementation of the server interface.
// This will handle the identity message sent by the client.
type serverListener struct {
	// connectionCallback allows an AuthenticatedConnection
	// to be passed back upon establishment.
	connectionCallback AuthenticatedCallback

	// conn used to retrieve the connection context with the partner.
	conn Connection
}

// buildAuthConfirmationHandler returns a serverListener object.
// This will handle incoming identity authentication confirmations
// via the serverListener.Hear method. A successful AuthenticatedConnection
// will be passed along via the serverListener.connectionCallback.
func buildAuthConfirmationHandler(cb AuthenticatedCallback,
	connection Connection) server {
	return &serverListener{
		connectionCallback: cb,
		conn:               connection,
	}
}

// Hear handles the reception of an IdentityAuthentication by the
// server. It will attempt to verify the identity confirmation of
// the given client.
func (a serverListener) Hear(item receive.Message) {
	// Process the message data into a protobuf
	iar := &IdentityAuthentication{}
	err := proto.Unmarshal(item.Payload, iar)
	if err != nil {
		a.handleAuthConfirmationErr(err, item.Sender)
		return
	}

	// Get the new partner
	newPartner := a.conn.GetPartner()
	connectionFp := newPartner.ConnectionFingerprint().Bytes()

	// Verify the signature within the message
	err = connCrypto.Verify(newPartner.PartnerId(),
		iar.Signature, connectionFp, iar.RsaPubKey, iar.Salt)
	if err != nil {
		a.handleAuthConfirmationErr(err, item.Sender)
		return
	}

	// If successful, pass along the established authenticated connection
	// via the callback
	jww.DEBUG.Printf("AuthenticatedConnection auth request for %s confirmed",
		item.Sender.String())
	authConn := BuildAuthenticatedConnection(a.conn)
	authConn.setAuthenticated()
	go a.connectionCallback(authConn)
}

// handleAuthConfirmationErr is a helper function which will close the connection
// between the server and the client. It will also print out the passed in error.
func (a serverListener) handleAuthConfirmationErr(err error, sender *id.ID) {
	jww.ERROR.Printf("Unable to build connection with "+
		"partner %s: %+v", sender, err)
	// Send a nil connection to avoid hold-ups down the line
	a.connectionCallback(nil)
	err = a.conn.Close()
	if err != nil {
		jww.ERROR.Printf("Failed to close connection with partner %s: %v",
			sender, err)
	}
}

// Name returns the name of this listener. This is typically for
// printing/debugging purposes.
func (a serverListener) Name() string {
	return authenticatedServerListenerName
}
