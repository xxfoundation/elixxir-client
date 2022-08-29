////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/parse"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/stoppable"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"testing"
	"time"
)

func Test_manager_SendE2E_Smoke(t *testing.T) {
	streamGen := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG)
	rng := streamGen.GetStream()
	defer rng.Close()
	netHandler := newMockCmixHandler()

	// Generate new E2E manager
	myKv := versioned.NewKV(ekv.MakeMemstore())
	myID := id.NewIdFromString("myID", id.User, t)
	myNet := newMockCmix(myID, netHandler, t)
	m1 := &manager{
		Switchboard: receive.New(),
		partitioner: parse.NewPartitioner(myKv, myNet.GetMaxMessageLength()),
		net:         myNet,
		myID:        myID,
		events:      mockEventsManager{},
		grp:         myNet.GetInstance().GetE2EGroup(),
		rekeyParams: rekey.GetDefaultParams(),
	}

	myPrivKey := dh.GeneratePrivateKey(
		dh.DefaultPrivateKeyLength, m1.grp, rng)
	err := ratchet.New(myKv, myID, myPrivKey, m1.grp)
	if err != nil {
		t.Errorf("Failed to generate new ratchet: %+v", err)
	}

	myFpGen := &fpGenerator{m1}
	myServices := newMockServices()

	m1.Ratchet, err = ratchet.Load(
		myKv, myID, m1.grp, myFpGen, myServices, streamGen)

	// Generate new E2E manager
	partnerKv := versioned.NewKV(ekv.MakeMemstore())
	partnerID := id.NewIdFromString("partnerID", id.User, t)
	partnerNet := newMockCmix(partnerID, netHandler, t)
	m2 := &manager{
		Switchboard: receive.New(),
		partitioner: parse.NewPartitioner(partnerKv, partnerNet.GetMaxMessageLength()),
		net:         partnerNet,
		myID:        partnerID,
		events:      mockEventsManager{},
		grp:         partnerNet.GetInstance().GetE2EGroup(),
		rekeyParams: rekey.GetDefaultParams(),
	}

	receiveChan := make(chan receive.Message, 10)
	m2.Switchboard.RegisterListener(partnerID, catalog.NoType, &mockListener{receiveChan})

	partnerPrivKey := dh.GeneratePrivateKey(
		dh.DefaultPrivateKeyLength, m2.grp, rng)
	err = ratchet.New(partnerKv, partnerID, partnerPrivKey, m2.grp)
	if err != nil {
		t.Errorf("Failed to generate new ratchet: %+v", err)
	}

	partnerFpGen := &fpGenerator{m2}
	partnerServices := newMockServices()

	m1.Ratchet, err = ratchet.Load(
		partnerKv, partnerID, m2.grp, partnerFpGen, partnerServices, streamGen)

	// Generate partner identity and add partner
	partnerPubKey, partnerSidhPubKey, mySidhPrivKey, sessionParams :=
		genPartnerKeys(partnerPrivKey, m1.grp, rng, t)
	_, err = m1.Ratchet.AddPartner(partnerID, partnerPubKey, myPrivKey,
		partnerSidhPubKey, mySidhPrivKey, sessionParams, sessionParams)
	if err != nil {
		t.Errorf("Failed to add partner: %+v", err)
	}

	payload := []byte("My Payload")
	p := GetDefaultParams()
	_, err = m1.SendE2E(catalog.NoType, partnerID, payload, p)
	if err != nil {
		t.Errorf("SendE2E failed: %+v", err)
	}

	select {
	case r := <-receiveChan:
		if !bytes.Equal(payload, r.Payload) {
			t.Errorf("Received payload does not match sent payload."+
				"\nexpected: %q\nreceived: %q", payload, r.Payload)
		}
	case <-time.After(305 * time.Millisecond):
		t.Errorf("Timed out waiting for E2E message.")
	}
}

// genPartnerKeys generates the keys needed to add a partner.
func genPartnerKeys(partnerPrivKey *cyclic.Int, grp *cyclic.Group,
	rng io.Reader, t testing.TB) (
	partnerPubKey *cyclic.Int, partnerSidhPubKey *sidh.PublicKey,
	mySidhPrivKey *sidh.PrivateKey, params session.Params) {

	partnerPubKey = dh.GeneratePublicKey(partnerPrivKey, grp)

	partnerSidhPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSidhPubKey = util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	err := partnerSidhPrivKey.Generate(rng)
	if err != nil {
		t.Fatalf("Failed to generate partner SIDH private key: %+v", err)
	}
	partnerSidhPrivKey.GeneratePublicKey(partnerSidhPubKey)

	mySidhPrivKey = util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySidhPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	err = mySidhPrivKey.Generate(rng)
	if err != nil {
		t.Fatalf("Failed to generate my SIDH private key: %+v", err)
	}
	mySidhPrivKey.GeneratePublicKey(mySidhPubKey)

	params = session.GetDefaultParams()

	return partnerPubKey, partnerSidhPubKey, mySidhPrivKey, params
}

// Tests that waitForKey returns a key after it waits 5 times.
func Test_waitForKey(t *testing.T) {
	wait := 15 * time.Millisecond
	numAttempts := uint(10)
	expectedCypher := &mockWaitForKeyCypher{5}
	var attempt uint
	keyGetter := func() (session.Cypher, error) {
		if attempt >= (numAttempts / 2) {
			return expectedCypher, nil
		}
		attempt++
		return nil, errors.New("Failed to get key.")
	}
	stop := stoppable.NewSingle("Test_waitForKey")

	c, err := waitForKey(keyGetter, numAttempts, wait, stop, &id.ID{}, "", 0)
	if err != nil {
		t.Errorf("waitForKey returned an error: %+v", err)
	}

	if *c.(*mockWaitForKeyCypher) != *expectedCypher {
		t.Errorf("Received unexpected cypher.\nexpected: %#v\nreceived: %#v",
			*expectedCypher, *c.(*mockWaitForKeyCypher))
	}
}

// Error path: tests that waitForKey returns an error after the key getter does
// not return any keys after all attempts
func Test_waitForKey_NoKeyError(t *testing.T) {
	expectedErr := "Failed to get key."
	keyGetter := func() (session.Cypher, error) {
		return nil, errors.New(expectedErr)
	}
	stop := stoppable.NewSingle("Test_waitForKey")

	_, err := waitForKey(keyGetter, 10, 1, stop, &id.ID{}, "", 0)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("waitForKey did not return the expected error when no key "+
			"is available.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: tests that waitForKey returns an error after the stoppable is
// triggered.
func Test_waitForKey_StopError(t *testing.T) {
	expectedErr := "Stopped by stopper"
	keyGetter := func() (session.Cypher, error) {
		return nil, errors.New("Failed to get key.")
	}
	stop := stoppable.NewSingle("Test_waitForKey")

	go func() {
		_, err := waitForKey(keyGetter, 10, 1, stop, &id.ID{}, "", 0)
		if err == nil || err.Error() != expectedErr {
			t.Errorf("waitForKey did not return the expected error when the "+
				"stoppable was triggered.\nexpected: %s\nreceived: %+v",
				expectedErr, err)
		}
	}()

	err := stop.Close()
	if err != nil {
		t.Errorf("Failed to stop stoppable: %+v", err)
	}
}

type mockWaitForKeyCypher struct {
	cypherNum int
}

func (m *mockWaitForKeyCypher) GetSession() *session.Session    { return nil }
func (m *mockWaitForKeyCypher) Fingerprint() format.Fingerprint { return format.Fingerprint{} }
func (m *mockWaitForKeyCypher) Encrypt([]byte) ([]byte, []byte, e2e.KeyResidue) {
	return nil, nil, e2e.KeyResidue{}
}
func (m *mockWaitForKeyCypher) Decrypt(format.Message) ([]byte, e2e.KeyResidue, error) {
	return nil, e2e.KeyResidue{}, nil
}
func (m *mockWaitForKeyCypher) Use() {}

// Tests that getSendErrors returns all the errors on the channel.
func Test_getSendErrors(t *testing.T) {
	const n = 10
	var expectedErrors string
	errorList := make([]error, n)
	for i := range errorList {
		errorList[i] = errors.Errorf("Error %d of %d", i, n)
		expectedErrors += errorList[i].Error()
	}

	c := make(chan error, n*2)
	for _, e := range errorList {
		c <- e
	}

	numFail, errRtn := getSendErrors(c)

	if numFail != n {
		t.Errorf("Incorrect number of failed.\nexpected: %d\nreceived: %d",
			n, numFail)
	}

	if errRtn != expectedErrors {
		t.Errorf("Received incorrect errors.\nexpected: %q\nreceived: %q",
			expectedErrors, errRtn)
	}
}
