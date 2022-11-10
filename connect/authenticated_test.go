////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"math/rand"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// TestConnectWithAuthentication will test the client/server relationship for
// an AuthenticatedConnection. This will construct a client which will send an
// IdentityAuthentication message to the server, who will hear it and verify
// the contents. This will use a mock connection interface and private
// production code helper functions for easier testing.
func TestConnectWithAuthentication(t *testing.T) {
	grp := getGroup()
	numPrimeByte := len(grp.GetPBytes())

	// Set up cmix handler
	mockNet := newMockCmix()

	// Set up connect arguments
	prng := rand.New(rand.NewSource(42))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		numPrimeByte, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)
	salt := make([]byte, 32)
	copy(salt, "salt")

	myRsaPrivKey, err := rsa.LoadPrivateKeyFromPem(getPrivKey())
	if err != nil {
		t.Fatalf("Failed to load private key: %v", err)
	}

	// Construct client ID the proper way as server will need to verify it
	// using the xx.NewID function call
	myId, err := xx.NewID(myRsaPrivKey.GetPublic(), salt, id.User)
	if err != nil {
		t.Fatalf("Failed to generate client's id: %+v", err)
	}

	// Generate server ID using testing interface
	serverID := id.NewIdFromString("server", id.User, t)

	// Construct recipient
	recipient := contact.Contact{
		ID:       serverID,
		DhPubKey: dhPubKey,
	}

	rng := fastRNG.NewStreamGenerator(1, 1,
		csprng.NewSystemRNG)

	// Create the mock connection, which will be shared by the client and
	// server. This will send the client's request to the server internally
	mockConn := newMockConnection(myId, serverID, dhPrivKey, dhPubKey)

	// Set up the server's callback, which will pass the authenticated
	// connection through via a channel
	authConnChan := make(chan AuthenticatedConnection, 1)
	serverCb := AuthenticatedCallback(
		func(connection AuthenticatedConnection) {
			authConnChan <- connection
		})

	// Initialize params with a shorter timeout to hasten test results
	customParams := xxdk.GetDefaultE2EParams()
	customParams.Base.Timeout = 3 * time.Second

	// Initialize the server
	serverHandler := buildAuthConfirmationHandler(serverCb, mockConn)

	// Pass the server's listener to the mock connection so the connection
	// can pass the client's message directly to the server
	mockConn.listener = serverHandler

	// Initialize the client
	_, err = connectWithAuthentication(mockConn, time.Now(), recipient,
		salt, myRsaPrivKey, rng, mockNet,
		customParams)
	if err != nil {
		t.Fatalf("ConnectWithAuthentication error: %+v", err)
	}

	// Wait for the server to establish it's connection via the callback
	timeout := time.NewTimer(customParams.Base.Timeout)
	select {
	case <-authConnChan:
		return
	case <-timeout.C:
		t.Fatalf("Timed out waiting for server's authenticated connection " +
			"to be established")
	}

}
