package authenticated

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/connections/connect"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/crypto/signature/rsa"
)

// clientListenerName is the name of the client's listener interface.
const clientListenerName = "AuthenticatedClientListener"

// client is an interface that collates the receive.Listener and
// auth.Callbacks interfaces. This allows a singular object
// to handle auth package endpoints as well as handling the
// custom catalog.MessageType's.
type client interface {
	receive.Listener
	auth.Callbacks
}

// clientHandler provides an implementation of the client interface.
type clientHandler struct {
	// connectionCallback allows an authenticated.Connection
	// to be passed back upon establishment.
	connectionCallback ConnectionCallback

	// Used for building new Connection objects
	connectionE2e    clientE2e.Handler
	connectionParams connect.Params
}

// getClient returns a client interface to be used to handle
// auth.Callbacks and receive.Listener operations.
func getClient(cb ConnectionCallback,
	e2e clientE2e.Handler, p connect.Params) client {
	return clientHandler{
		connectionCallback: cb,
		connectionE2e:      e2e,
		connectionParams:   p,
	}
}

// Request will be called when an auth Request message is processed.
func (a clientHandler) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// Confirm will be called when an auth Confirm message is processed.
func (a clientHandler) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// Reset will be called when an auth Reset operation occurs.
func (a clientHandler) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// Hear handles the reception of an IdentityAuthentication by the
// server.
func (a clientHandler) Hear(item receive.Message) {
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
	newPartner, err := a.connectionE2e.GetPartner(item.Sender)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	// The connection fingerprint (hashed) represents a shared nonce
	// between these two partners
	conneptionFp := newPartner.ConnectionFingerprint().Bytes()

	// Hash the connection fingerprint
	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(conneptionFp)
	nonce := h.Sum(nil)

	// Verify the signature
	err = rsa.Verify(partnerPubKey, opts.Hash, nonce, iar.Signature, opts)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", item.Sender, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

	// If successful, pass along the established connection
	jww.DEBUG.Printf("Connection auth request for %s confirmed",
		item.Sender.String())
	a.connectionCallback(buildAuthenticatedConnection(newPartner, a.connectionE2e,
		a.connectionParams))
}

func (a clientHandler) Name() string {
	return clientListenerName
}
