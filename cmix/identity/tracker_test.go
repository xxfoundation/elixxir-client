////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package identity

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/address"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"testing"
	"time"
)

func TestManager_processIdentities_expired(t *testing.T) {
	// Initialization
	addrSpace := address.NewAddressSpace()
	addrSpace.UpdateAddressSpace(18)
	session := storage.InitTestingSession(t)
	m := &manager{
		tracked:        make([]TrackedID, 0),
		session:        session,
		newIdentity:    make(chan TrackedID, trackedIDChanSize),
		deleteIdentity: make(chan *id.ID, deleteIDChanSize),
		addrSpace:      addrSpace,
		ephemeral:      receptionID.NewOrLoadStore(session.GetKV()),
		mux:            &sync.Mutex{},
	}

	// Add some expired test IDs
	for i := uint64(0); i < 10; i++ {
		testId := id.NewIdFromUInt(i, id.User, t)
		validUntil := time.Now()
		m.tracked = append(m.tracked, TrackedID{
			NextGeneration: netTime.Now().Add(-time.Second),
			LastGeneration: time.Time{},
			Source:         testId,
			ValidUntil:     validUntil,
			Persistent:     false,
			Creation:       netTime.Now(),
		})
	}

	expected := m.tracked[0].ValidUntil
	nextEvent := m.processIdentities(addrSpace.GetAddressSpace())
	if len(m.tracked) != 0 {
		t.Errorf("Failed to remove expired identities, %d remain", len(m.tracked))
	}
	if nextEvent != expected {
		t.Errorf("Invalid nextEvent, expected %v got %v", expected, nextEvent)
	}
}

func TestManager_processIdentities(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	// Initialization
	addrSpace := address.NewAddressSpace()
	addrSpace.UpdateAddressSpace(18)
	session := storage.InitTestingSession(t)
	m := &manager{
		tracked:        make([]TrackedID, 0),
		session:        session,
		newIdentity:    make(chan TrackedID, trackedIDChanSize),
		deleteIdentity: make(chan *id.ID, deleteIDChanSize),
		addrSpace:      addrSpace,
		ephemeral:      receptionID.NewOrLoadStore(session.GetKV()),
		mux:            &sync.Mutex{},
	}

	// Add some expired test IDs
	testId := id.NewIdFromUInt(0, id.User, t)
	validUntil := time.Now().Add(time.Minute)
	m.tracked = append(m.tracked, TrackedID{
		NextGeneration: netTime.Now(),
		LastGeneration: time.Time{},
		Source:         testId,
		ValidUntil:     validUntil,
		Persistent:     true,
		Creation:       netTime.Now(),
	})

	_ = m.processIdentities(addrSpace.GetAddressSpace())
	if len(m.tracked) != 1 {
		t.Errorf("Unexpectedly removed identity, %d remain", len(m.tracked))
	}
}
