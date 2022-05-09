///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package authenticated

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/connections/connect"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// serverListenerName is the name of the client's listener interface.
const serverListenerName = "AuthenticatedServerListener"

// server is an interface that wraps receive.Listener. This handles
// the server listening for the client's proof of identity message.
type server interface {
	receive.Listener
}

// serverListener provides an implementation of the server interface.
// This will handle the identity message sent by the client.
type serverListener struct {
	// connectionCallback allows an authenticated.Connection
	// to be passed back upon establishment.
	connectionCallback ConnectionCallback

	// conn used to retrieve the connection context with the partner.
	conn connect.Connection
}

// handleAuthConfirmation returns a serverListener object.
func handleAuthConfirmation(cb ConnectionCallback,
	connection connect.Connection) server {
	return serverListener{
		connectionCallback: cb,
		conn:               connection,
	}
}

// Hear handles the reception of an IdentityAuthentication by the
// server.
func (a serverListener) Hear(item receive.Message) {
	// Process the message data into a protobuf
	iar := &IdentityAuthentication{}
	err := proto.Unmarshal(item.Payload, iar)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	// Process the PEM encoded public key to an rsa.PublicKey object
	partnerPubKey, err := rsa.LoadPublicKeyFromPem(iar.RsaPubKey)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	// Get the new partner
	newPartner := a.conn.GetPartner()

	// Verify the partner's known ID against the information passed
	// along the wire
	partnerWireId, err := xx.NewID(partnerPubKey, iar.Salt, id.User)
	if err != nil {
		jww.ERROR.Printf("Unable to parse identity information with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	if !newPartner.PartnerId().Cmp(partnerWireId) {
		jww.ERROR.Printf("Unable to verify identity information with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	// The connection fingerprint (hashed) represents a shared nonce
	// between these two partners
	connectionFp := newPartner.ConnectionFingerprint().Bytes()

	// Hash the connection fingerprint
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Verify the signature
	err = rsa.Verify(partnerPubKey, opts.Hash, nonce, iar.Signature, opts)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

	// If successful, pass along the established authenticated connection
	// via the callback
	jww.DEBUG.Printf("Connection auth request for %s confirmed",
		item.Sender.String())
	authConn := buildAuthenticatedConnection(a.conn)
	authConn.setAuthenticated()
	a.connectionCallback(authConn)
}

// Name returns the name of this listener. This is typically for
// printing/debugging purposes.
func (a serverListener) Name() string {
	return serverListenerName
}
