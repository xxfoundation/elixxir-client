package authenticated

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/connections/connect"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// initiateClientAuthentication is a helper function which will handle the
// establishment of an authenticated.Connection. This will have the
// client send an identity authentication message to the server. Upon
// successful sending of the message (determined by the result of the round(s))
// an authenticated.Connection is assumed to be established and passed along
// via the callback.
func initiateClientAuthentication(authCb ConnectionCallback,
	conn connect.Connection, net cmix.Client,
	rng *fastRNG.StreamGenerator, rsaPrivKey *rsa.PrivateKey, salt []byte,
	connParams connect.Params) {

	// After confirmation, get the new partner
	newPartner := conn.GetPartner()

	// The connection fingerprint (hashed) represents a shared nonce
	// between these two partners
	connectionFp := newPartner.ConnectionFingerprint().Bytes()

	opts := rsa.NewDefaultOptions()
	h := opts.Hash.New()
	h.Write(connectionFp)
	nonce := h.Sum(nil)

	// Sign the connection fingerprint
	stream := rng.GetStream()
	defer stream.Close()
	signature, err := rsa.Sign(stream, rsaPrivKey,
		opts.Hash, nonce, opts)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", newPartner.PartnerId(), err)
		// Send a nil connection to avoid hold-ups down the line
		authCb(nil)
	}

	// Construct message
	pemEncodedRsaPubKey := rsa.CreatePublicKeyPem(rsaPrivKey.GetPublic())
	iar := &IdentityAuthentication{
		Signature: signature,
		RsaPubKey: pemEncodedRsaPubKey,
		Salt:      salt,
	}
	payload, err := proto.Marshal(iar)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", newPartner.PartnerId(), err)
		// Send a nil connection to avoid hold-ups down the line
		authCb(nil)
	}

	// Send message to user
	rids, _, _, err := conn.SendE2E(catalog.ConnectionAuthenticationRequest,
		payload, clientE2e.GetDefaultParams())
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", newPartner.PartnerId(), err)
		// Send a nil connection to avoid hold-ups down the line
		authCb(nil)
	}

	// Determine that the message is properly sent by tracking the success
	// of the round(s)
	roundCb := cmix.RoundEventCallback(func(allRoundsSucceeded,
		timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		if allRoundsSucceeded {
			// If rounds succeeded, assume recipient has successfully
			// confirmed the authentication. Pass the connection
			// along via the callback
			authConn := buildAuthenticatedConnection(conn)
			authConn.setAuthenticated()
			authCb(authConn)
		} else {
			jww.ERROR.Printf("Unable to build connection with "+
				"partner %s: %+v", newPartner.PartnerId(), err)
			// Send a nil connection to avoid hold-ups down the line
			authCb(nil)
		}
	})

	err = net.GetRoundResults(connParams.Timeout,
		roundCb, rids...)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", newPartner.PartnerId(), err)
		// Send a nil connection to avoid hold-ups down the line
		authCb(nil)
	}

}
