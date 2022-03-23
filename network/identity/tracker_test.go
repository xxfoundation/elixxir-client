///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package identity

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	ephemeral2 "gitlab.com/elixxir/client/network/address"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
	"time"
)

// Smoke test for Track function
func TestCheck(t *testing.T) {
	session := storage.InitTestingSession(t)
	instance := ephemeral2.NewTestNetworkManager(t)
	if err := setupInstance(instance); err != nil {
		t.Errorf("Could not set up instance: %+v", err)
	}

	// Store a mock initial timestamp the store
	now := netTime.Now()
	twoDaysAgo := now.Add(-2 * 24 * time.Hour)
	twoDaysTimestamp, err := marshalTimestamp(twoDaysAgo)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %+v", err)
	}

	err = session.Set(TimestampKey, twoDaysTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %+v", err)
	}

	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := Track(session, ephemeral2.NewTestAddressSpace(15, t), ourId)

	err = stop.Close()
	if err != nil {
		t.Errorf("Could not close thread: %+v", err)
	}

}

// Unit test for track.
func TestCheck_Thread(t *testing.T) {
	session := storage.InitTestingSession(t)
	instance := ephemeral2.NewTestNetworkManager(t)
	if err := setupInstance(instance); err != nil {
		t.Errorf("Could not set up instance: %v", err)
	}
	ourId := id.NewIdFromBytes([]byte("Sauron"), t)
	stop := stoppable.NewSingle(ephemeralStoppable)

	// Store a mock initial timestamp the store
	now := netTime.Now()
	yesterday := now.Add(-24 * time.Hour)
	yesterdayTimestamp, err := marshalTimestamp(yesterday)
	if err != nil {
		t.Errorf("Could not marshal timestamp for test setup: %+v", err)
	}

	err = session.Set(TimestampKey, yesterdayTimestamp)
	if err != nil {
		t.Errorf("Could not set mock timestamp for test setup: %+v", err)
	}

	// Run the tracker
	go func() {
		track(session, ephemeral2.NewTestAddressSpace(15, t), ourId, stop)
	}()
	time.Sleep(3 * time.Second)

	err = stop.Close()
	if err != nil {
		t.Errorf("Could not close thread: %v", err)
	}

}

func setupInstance(instance interfaces.NetworkManager) error {
	cert, err := utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		return errors.Errorf("Failed to read cert from from file: %+v", err)
	}
	ri := &mixmessages.RoundInfo{
		ID: 1,
	}

	testCert, err := rsa.LoadPrivateKeyFromPem(cert)
	if err != nil {
		return errors.Errorf("Failed to load cert from from file: %+v", err)
	}
	if err = signature.SignRsa(ri, testCert); err != nil {
		return errors.Errorf("Failed to sign round info: %+v", err)
	}
	if _, err = instance.GetInstance().RoundUpdate(ri); err != nil {
		return errors.Errorf("Failed to RoundUpdate from from file: %+v", err)
	}

	ri = &mixmessages.RoundInfo{
		ID: 2,
	}
	if err = signature.SignRsa(ri, testCert); err != nil {
		return errors.Errorf("Failed to sign round info: %+v", err)
	}
	if _, err = instance.GetInstance().RoundUpdate(ri); err != nil {
		return errors.Errorf("Failed to RoundUpdate from from file: %v", err)
	}

	return nil
}

func TestGenerateIdentities(t *testing.T) {
	eid, s, e, err := ephemeral.GetId(id.NewIdFromString("zezima", id.Node, t), 16, time.Now().UnixNano())
	if err != nil {
		t.Errorf("Failed to get eid: %+v", err)
	}
	protoIds := []ephemeral.ProtoIdentity{{eid, s, e}}
	generated := generateIdentities(protoIds, id.NewIdFromString("escaline", id.Node, t), 16)
	if generated[0].EndValid != protoIds[0].End.Add(5*time.Minute) {
		t.Errorf("End was not modified.  Orig %+v, Generated %+v", protoIds[0].End, generated[0].End)
	}
	if generated[0].StartValid != protoIds[0].Start.Add(-5*time.Minute) {
		t.Errorf("End was not modified.  Orig %+v, Generated %+v", protoIds[0].End, generated[0].End)
	}
}
