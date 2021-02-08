///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ephemeral

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"
)

// Smoke test for Track function
func TestCheck(t *testing.T) {
	session := storage.InitTestingSession(t)
	instance := NewTestNetworkManager(t)
	if err := setupInstance(instance); err != nil {
		t.Errorf("Could not set up instance: %v", err)
	}

	/// Store a mock initial timestamp the store
	now := time.Now()
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	twoDaysTimestamp, err := marshalTimestamp(twoDaysAgo)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %v", err)
	}
	err = session.Set(TimestampKey, twoDaysTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %v", err)
	}

	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := Track(session, instance.GetInstance(), ourId)

	err = stop.Close(3 * time.Second)
	if err != nil {
		t.Errorf("Could not close thread: %v", err)
	}

}

// Unit test for track
func TestCheck_Thread(t *testing.T) {

	session := storage.InitTestingSession(t)
	instance := NewTestNetworkManager(t)
	if err := setupInstance(instance); err != nil {
		t.Errorf("Could not set up instance: %v", err)
	}
	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := stoppable.NewSingle(ephemeralStoppable)

	/// Store a mock initial timestamp the store
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	yesterdayTimestamp, err := marshalTimestamp(yesterday)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %v", err)
	}
	err = session.Set(TimestampKey, yesterdayTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %v", err)
	}

	// Run the tracker
	go func() {
		track(session, instance.GetInstance(), ourId, stop)
	}()

	time.Sleep(1 * time.Second)

	// Manually generate identities

	eids, err := ephemeral.GetIdsByRange(ourId, session.Reception().GetIDSize(), now.UnixNano(), now.Sub(yesterday))
	if err != nil {
		t.Errorf("Could not generate upcoming ids: %v", err)
	}

	identities := generateIdentities(eids, ourId)

	rngStreamGen := fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG)

	retrieved, err := session.Reception().GetIdentity(rngStreamGen.GetStream())
	if err != nil {
		t.Errorf("Could not retrieve identity: %v", err)
	}

	// Check if store has been updated for new identities
	if identities[0].String() != retrieved.String() {
		t.Errorf("Store was not updated for newly generated identies")
	}

	err = stop.Close(3 * time.Second)
	if err != nil {
		t.Errorf("Could not close thread: %v", err)
	}

}

func setupInstance(instance interfaces.NetworkManager) error {
	cert, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		return errors.Errorf("Failed to read cert from from file: %v", err)
	}
	ri := &mixmessages.RoundInfo{
		ID:               1,
		AddressSpaceSize: 64,
	}

	testCert, err := rsa.LoadPrivateKeyFromPem(cert)
	if err != nil {
		return errors.Errorf("Failed to load cert from from file: %v", err)
	}
	if err = signature.Sign(ri, testCert); err != nil {
		return errors.Errorf("Failed to sign round info: %v", err)
	}
	if err = instance.GetInstance().RoundUpdate(ri); err != nil {
		return errors.Errorf("Failed to RoundUpdate from from file: %v", err)
	}

	ri = &mixmessages.RoundInfo{
		ID:               2,
		AddressSpaceSize: 64,
	}
	if err = signature.Sign(ri, testCert); err != nil {
		return errors.Errorf("Failed to sign round info: %v", err)
	}
	if err = instance.GetInstance().RoundUpdate(ri); err != nil {
		return errors.Errorf("Failed to RoundUpdate from from file: %v", err)
	}

	return nil
}
