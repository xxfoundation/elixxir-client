////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"gitlab.com/elixxir/client/v5/catalog"
	"gitlab.com/elixxir/client/v5/e2e/parse"
	"gitlab.com/elixxir/client/v5/e2e/ratchet"
	"gitlab.com/elixxir/client/v5/e2e/receive"
	"gitlab.com/elixxir/client/v5/e2e/rekey"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestManager_SendUnsafe(t *testing.T) {
	streamGen := fastRNG.NewStreamGenerator(12,
		1024, csprng.NewSystemRNG)
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
	m2.Switchboard.RegisterListener(myID, catalog.NoType, &mockListener{receiveChan})

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

	m1.EnableUnsafeReception()
	m2.EnableUnsafeReception()

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

	_, _, err = m1.sendUnsafe(catalog.NoType, partnerID, payload, p)
	if err != nil {
		t.Fatalf("sendUnsafe error: %v", err)
	}
}
