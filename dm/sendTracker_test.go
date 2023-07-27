////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// Test MessageReceive basic logic.
func TestSendTracker_MessageReceive(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uuidNum := uint64(0)
	rid := id.Round(2)

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()

	r := rounds.Round{
		ID:         rid,
		Timestamps: make(map[states.Round]time.Time),
	}
	r.Timestamps[states.QUEUED] = time.Now()
	trigger := func(msgID message.ID, messageType MessageType,
		nick string, plaintext []byte, dmToken uint32,
		partnerPubKey, senderKey ed25519.PublicKey, ts time.Time,
		_ receptionID.EphemeralIdentity, round rounds.Round,
		status Status) (uint64, error) {
		oldUUID := uuidNum
		uuidNum++
		return oldUUID, nil
	}

	updateStatus := func(uuid uint64, messageID message.ID,
		timestamp time.Time, round rounds.Round, status Status) {
	}

	cid := id.NewIdFromString("channel", id.User, t)

	st := NewSendTracker(kv)
	st.Init(&mockClient{}, trigger, updateStatus, crng)

	directMessage := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}
	mid := message.DeriveDirectMessageID(cid, directMessage)
	process := st.CheckIfSent(mid, r)
	require.False(t, process)

	uuid, err := st.DenotePendingSend(partner.PubKey, me.PubKey,
		partner.GetDMToken(), 0, directMessage)
	require.NoError(t, err)

	err = st.Sent(uuid, mid, rounds.Round{
		ID:    rid,
		State: 1,
	})
	require.NoError(t, err)

	process = st.CheckIfSent(mid, r)
	st.Delivered(mid, r)
	st.StopTracking(mid, r)
	require.True(t, process)

	directMessage2 := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello again"),
	}
	uuid2, err := st.DenotePendingSend(partner.PubKey, me.PubKey,
		partner.GetDMToken(), 0, directMessage2)
	require.NoError(t, err)

	err = st.Sent(uuid2, mid, rounds.Round{
		ID:    rid,
		State: 1,
	})
	require.NoError(t, err)
	process = st.CheckIfSent(mid, r)
	require.True(t, process)
	st.Delivered(mid, r)
	st.StopTracking(mid, r)
}

// Test failedSend function, confirming that data is stored appropriately and
// callbacks are called.
func TestSendTracker_failedSend(t *testing.T) {
	triggerCh := make(chan Status)

	kv := versioned.NewKV(ekv.MakeMemstore())

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()
	partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&partner.PubKey)

	partnerID := deriveReceptionID(partnerPubKey.Bytes(),
		partner.GetDMToken())

	updateStatus := func(uuid uint64, messageID message.ID,
		timestamp time.Time, round rounds.Round, status Status) {
		triggerCh <- status
	}

	st := &sendTracker{kv: kv}
	st.Init(&mockClient{}, emptyTrigger, updateStatus, crng)

	rid := id.Round(2)
	directMessage := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}
	mid := message.DeriveDirectMessageID(partnerID, directMessage)
	uuid, err := st.DenotePendingSend(partner.PubKey, me.PubKey,
		partner.GetDMToken(), 0, directMessage)
	require.NoError(t, err)

	err = st.FailedSend(uuid)
	require.NoError(t, err)

	timeout := time.NewTicker(time.Second * 5)
	select {
	case s := <-triggerCh:
		if s != Failed {
			t.Fatalf("Did not receive failed from failed message")
		}
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	require.False(t, ok)
	require.Equal(t, len(trackedRound.List), 0)

	_, ok = st.byMessageID[mid]
	require.False(t, ok)

	_, ok = st.unsent[uuid]
	require.False(t, ok)
}

// Test send tracker send function, confirming that data is stored appropriately
// // and callbacks are called
func TestSendTracker_send(t *testing.T) {
	triggerCh := make(chan bool)

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()
	partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&partner.PubKey)

	partnerID := deriveReceptionID(partnerPubKey.Bytes(),
		partner.GetDMToken())

	kv := versioned.NewKV(ekv.MakeMemstore())

	updateStatus := func(uuid uint64, messageID message.ID,
		timestamp time.Time, round rounds.Round, status Status) {
		triggerCh <- true
	}

	st := &sendTracker{kv: kv}
	st.Init(&mockClient{}, emptyTrigger, updateStatus, crng)

	rid := id.Round(2)
	directMessage := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}
	mid := message.DeriveDirectMessageID(partnerID, directMessage)
	uuid, err := st.DenotePendingSend(partner.PubKey, me.PubKey,
		partner.GetDMToken(), 0, directMessage)
	require.NoError(t, err)

	err = st.Sent(uuid, mid, rounds.Round{
		ID:    rid,
		State: 1,
	})
	require.NoError(t, err)

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	if !ok {
		t.Fatal("Should have found a tracked round")
	}
	require.Equal(t, len(trackedRound.List), 1)
	require.Equal(t, trackedRound.List[0].MsgID, mid)

	trackedMsg, ok := st.byMessageID[mid]
	require.True(t, ok)
	require.Equal(t, trackedMsg.MsgID, mid)
}

// Test loading stored byRound map from storage.
func TestSendTracker_load_store(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	// me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()
	partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&partner.PubKey)

	partnerID := deriveReceptionID(partnerPubKey.Bytes(),
		partner.GetDMToken())

	st := &sendTracker{kv: kv}
	st.Init(&mockClient{}, nil, nil, crng)

	rid := id.Round(2)
	directMessage := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}
	mid := message.DeriveDirectMessageID(partnerID, directMessage)
	st.byRound[rid] = trackedList{
		List: []*tracked{{MsgID: mid,
			partnerKey: partner.PubKey,
			RoundID:    rid}},
		RoundCompleted: false,
	}
	err := st.store()
	require.NoError(t, err)

	st2 := &sendTracker{kv: kv}
	st2.Init(&mockClient{}, nil, nil, crng)
	require.Equal(t, len(st2.byRound), len(st.byRound))
}

func TestRoundResult_callback(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	triggerCh := make(chan bool)
	update := func(uuid uint64, messageID message.ID,
		timestamp time.Time, round rounds.Round, status Status) {
		triggerCh <- true
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()
	partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&partner.PubKey)

	partnerID := deriveReceptionID(partnerPubKey.Bytes(),
		partner.GetDMToken())

	st := &sendTracker{kv: kv}
	st.Init(&mockClient{}, emptyTrigger, update, crng)

	rid := id.Round(2)
	directMessage := &DirectMessage{
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}
	mid := message.DeriveDirectMessageID(partnerID, directMessage)
	uuid, err := st.DenotePendingSend(partner.PubKey, me.PubKey,
		partner.GetDMToken(), 0, directMessage)
	require.NoError(t, err)

	err = st.Sent(uuid, mid, rounds.Round{
		ID:    rid,
		State: 2,
	})
	require.NoError(t, err)

	rr := roundResults{
		round:     rid,
		st:        st,
		numChecks: 0,
	}

	rr.callback(true, false, map[id.Round]cmix.RoundResult{
		rid: {Status: cmix.Succeeded, Round: rounds.Round{ID: rid,
			State: 0}}})

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
	case <-timeout.C:
		t.Fatal("Did not receive update")
	}
}

func emptyTrigger(msgID message.ID, messageType MessageType,
	nick string, plaintext []byte, dmToken uint32,
	partnerPubKey, senderKey ed25519.PublicKey, ts time.Time,
	_ receptionID.EphemeralIdentity, round rounds.Round,
	status Status) (uint64, error) {
	return 0, nil
}
