////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
	"time"
)

// Unit test for registerWithNode
func TestRegisterWithNodes(t *testing.T) {
	// Generate a stoppable
	stop := stoppable.NewSingle("test")
	defer stop.Quit()

	// Generate an rng
	stream := fastRNG.NewStreamGenerator(7, 3, csprng.NewSystemRNG).GetStream()
	defer stream.Close()

	/// Load private key
	privKeyRsa, err := rsa.LoadPrivateKeyFromPem([]byte(privKey))
	if err != nil {
		t.Fatalf("Failed to load private Key: %v", err)
	}

	// Initialize a fake node/gateway pair
	nid := id.NewIdFromString("ngw", id.Node, t)
	gwId := nid.DeepCopy()
	gwId.SetType(id.Gateway)
	ngw := network.NodeGateway{
		Node: ndf.Node{ID: nid.Bytes()},
		Gateway: ndf.Gateway{
			ID:             gwId.Bytes(),
			TlsCertificate: cert,
		},
	}

	// Initialize a mock time (not time.Now so that it can be constant)
	testTime, err := time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Could not parse precanned time: %v", err.Error())
	}

	// Initialize a mock session object
	salt := make([]byte, 32)
	copy(salt, "salt")
	mockSession := &mockSession{
		isPrecanned:     false,
		privKey:         privKeyRsa,
		timeStamp:       testTime,
		salt:            salt,
		transmissionSig: []byte("sig"),
	}

	//Generate an ephemeral DH key pair
	grp := getGroup()
	dhPriv := grp.RandomCoprime(grp.NewInt(1))

	// Initialize a mock comms
	secret := []byte("secret")
	mockComms := &MockClientComms{
		rsaPrivKey: privKeyRsa,
		dhPrivKey:  dhPriv,
		rand:       stream,
		secret:     secret,
		grp:        grp,
		t:          t,
	}
	r := makeTestRegistrar(mockComms, t)

	errCh := make(chan error)
	// Call registerWithNode
	go func() {
		errCh <- registerWithNodes([]network.NodeGateway{ngw},
			mockSession, r, stop)
	}()
	select {
	case <-r.rc:
	case err := <-errCh:
		t.Fatalf("registerWithNode error: %+v", err)
	}

}
