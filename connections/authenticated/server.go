package authenticated

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/connections/connect"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// serverHandler handles the io operations of an
// authenticated.Connection server.
type serverHandler struct {
	// connectionCallback allows an authenticated.Connection
	// to be passed back upon establishment.
	connectionCallback ConnectionCallback

	// Used for building new Connection objects
	connectionE2e    clientE2e.Handler
	connectionParams connect.Params

	// Used for tracking the round the identity confirmation message was sent.
	// A successful round assumes the client received the confirmation and
	// an authenticated.Connection has been established
	services cmix.Client

	// Used for signing the connection fingerprint used as a nonce
	// to confirm the user's identity
	privateKey *rsa.PrivateKey
	rng        *fastRNG.StreamGenerator
}

// getServer returns a serverHandler object. This is used to pass
// into the auth.State object to handle the auth.Callbacks.
func getServer(cb ConnectionCallback,
	e2e clientE2e.Handler, services cmix.Client,
	privateKey *rsa.PrivateKey,
	rng *fastRNG.StreamGenerator, p connect.Params) serverHandler {
	return serverHandler{
		connectionCallback: cb,
		connectionE2e:      e2e,
		connectionParams:   p,
		privateKey:         privateKey,
		rng:                rng,
		services:           services,
	}
}

// Request will be called when an auth Request message is processed.
func (a serverHandler) Request(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	// After confirmation, get the new partner
	newPartner, err := a.connectionE2e.GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
		return
	}

	authConn := buildAuthenticatedConnection(newPartner, a.connectionE2e,
		a.connectionParams)

	// The connection fingerprint (hashed) represents a shared nonce
	// between these two partners
	connectionFp := newPartner.ConnectionFingerprint().Bytes()

	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Sign the connection fingerprint
	stream := a.rng.GetStream()
	defer stream.Close()
	signature, err := rsa.Sign(stream, a.privateKey,
		opts.Hash, nonce, opts)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

	// Construct message
	pemEncodedRsaPubKey := rsa.CreatePublicKeyPem(a.privateKey.GetPublic())
	iar := &IdentityAuthentication{
		Signature: signature,
		RsaPubKey: pemEncodedRsaPubKey,
	}
	payload, err := proto.Marshal(iar)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

	// Send message to user
	rids, _, _, err := authConn.SendE2E(catalog.ConnectionAuthenticationRequest,
		payload, clientE2e.GetDefaultParams())
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

	// Determine that the message is properly sent by tracking the success
	// of the round(s)
	roundCb := cmix.RoundEventCallback(func(allRoundsSucceeded,
		timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		if allRoundsSucceeded {
			// If rounds succeeded, assume recipient has successfully
			// confirmed the authentication
			authConn.setAuthenticated()
			a.connectionCallback(authConn)
		} else {
			jww.ERROR.Printf("Unable to build connection with "+
				"partner %s: %+v", requestor.ID, err)
			// Send a nil connection to avoid hold-ups down the line
			a.connectionCallback(nil)
		}
	})
	err = a.services.GetRoundResults(a.connectionParams.Timeout,
		roundCb, rids...)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.connectionCallback(nil)
	}

}

// Confirm will be called when an auth Confirm message is processed.
func (a serverHandler) Confirm(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}

// Reset will be called when an auth Reset operation occurs.
func (a serverHandler) Reset(requestor contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
}
