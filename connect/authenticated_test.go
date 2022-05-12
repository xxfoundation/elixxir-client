///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
	"time"
)

// TestConnectWithAuthentication using the
func TestConnectWithAuthentication(t *testing.T) {
	grp := getGroup()
	numPrimeByte := len(grp.GetPBytes())

	cmixHandler := newMockCmixHandler()
	mockNet, err := newMockCmix(cmixHandler, t)
	if err != nil {
		t.Fatalf("Failed to initialize mock network: %+v", err)
	}
	prng := rand.New(rand.NewSource(42))
	dhPrivKey := diffieHellman.GeneratePrivateKey(
		numPrimeByte, grp, prng)
	dhPubKey := diffieHellman.GeneratePublicKey(dhPrivKey, grp)
	salt := make([]byte, 32)
	copy(salt, "salt")

	myRsaPrivKey, err := rsa.LoadPrivateKeyFromPem(getPrivKey())
	if err != nil {
		t.Fatalf("Faled to load private key: %v", err)
	}

	myId, err := xx.NewID(myRsaPrivKey.GetPublic(), salt, id.User)
	if err != nil {
		t.Fatalf("Failed to generate client's id: %+v", err)
	}
	serverID := id.NewIdFromString("server", id.User, t)

	recipient := contact.Contact{
		ID:       serverID,
		DhPubKey: dhPubKey,
	}

	rng := fastRNG.NewStreamGenerator(1, 1,
		csprng.NewSystemRNG)

	// Create the mock connection, which will be shared by the client and server.
	// This will send the client's request to the server internally
	mockConn := newMockConnection(myId, serverID, dhPrivKey, dhPubKey)

	// Set up the server
	authConnChan := make(chan AuthenticatedConnection, 1)
	serverCb := AuthenticatedCallback(
		func(connection AuthenticatedConnection) {
			authConnChan <- connection
		})

	customParams := GetDefaultParams()
	customParams.Timeout = 3 * time.Second

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

	timeout := time.NewTimer(customParams.Timeout)
	select {
	case <-authConnChan:
	case <-timeout.C:
		t.Fatalf("Timed out waiting for server's authenticated connection " +
			"to be established")
	}

}
