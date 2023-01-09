////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/xx_network/primitives/netTime"

	"github.com/golang/protobuf/proto"

	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

type triggerEventDummy struct {
	gotData bool

	chID        *id.ID
	umi         *userMessageInternal
	msgID       message.ID
	receptionID receptionID.EphemeralIdentity
	round       rounds.Round
}

func (ted *triggerEventDummy) triggerEvent(chID *id.ID,
	umi *userMessageInternal, _ []byte, _ time.Time,
	receptionID receptionID.EphemeralIdentity, round rounds.Round,
	_ SentStatus) (uint64, error) {
	ted.gotData = true

	ted.chID = chID
	ted.umi = umi
	ted.receptionID = receptionID
	ted.round = round
	ted.msgID = umi.GetMessageID()

	return 0, nil
}

// Tests the happy path.
func Test_userListener_Listen(t *testing.T) {

	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(42))
	pub, priv, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	cm := &ChannelMessage{
		Lease:       int64(time.Hour),
		RoundID:     uint64(r.ID),
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), cmSerial)

	sig := ed25519.Sign(priv, cmSerial)
	ns := &mockNameService{validChMsg: true}

	um := &UserMessage{
		Message:      cmSerial,
		Signature:    sig,
		ECCPublicKey: pub,
	}

	umSerial, err := proto.Marshal(um)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	// Build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:    chID,
		name:    ns,
		trigger: dummy.triggerEvent,
		checkSent: func(message.ID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(umSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if !dummy.gotData {
		t.Fatalf("No data returned after valid listen")
	}

	if !dummy.chID.Cmp(chID) {
		t.Errorf("Channel ID not correct: %s vs %s", dummy.chID, chID)
	}

	if !bytes.Equal(um.Message, dummy.umi.userMessage.Message) {
		t.Errorf("message not correct: %s vs %s", um.Message,
			dummy.umi.userMessage.Message)
	}

	if !msgID.Equals(dummy.msgID) {
		t.Errorf("messageIDs not correct: %s vs %s", msgID,
			dummy.msgID)
	}

	if r.ID != dummy.round.ID {
		t.Errorf("rounds not correct: %s vs %s", r.ID,
			dummy.round.ID)
	}
}

// Tests that the message is rejected when the user signature is invalid.
func Test_userListener_Listen_BadUserSig(t *testing.T) {
	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(42))
	pub, _, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	cm := &ChannelMessage{
		Lease:       int64(time.Hour),
		RoundID:     uint64(r.ID),
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	_, badPrivKey, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	sig := ed25519.Sign(badPrivKey, cmSerial)
	ns := &mockNameService{validChMsg: true}

	um := &UserMessage{
		Message:      cmSerial,
		Signature:    sig,
		ECCPublicKey: pub,
	}

	umSerial, err := proto.Marshal(um)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	// Build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:    chID,
		name:    ns,
		trigger: dummy.triggerEvent,
		checkSent: func(message.ID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(umSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

// Tests that the message is rejected when the round in the message does not
// match the round passed in.
func Test_userListener_Listen_BadRound(t *testing.T) {
	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(42))
	pub, priv, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	cm := &ChannelMessage{
		Lease:       int64(time.Hour),
		RoundID:     69, // Make the round not match
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	sig := ed25519.Sign(priv, cmSerial)
	ns := &mockNameService{validChMsg: true}

	um := &UserMessage{
		Message:      cmSerial,
		Signature:    sig,
		ECCPublicKey: pub,
	}

	umSerial, err := proto.Marshal(um)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	// Build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:    chID,
		name:    ns,
		trigger: dummy.triggerEvent,
		checkSent: func(message.ID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(umSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

// Tests that the message is rejected when the user message is malformed.
func Test_userListener_Listen_BadMessage(t *testing.T) {
	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	ns := &mockNameService{validChMsg: true}

	umSerial := []byte("malformed")

	// Build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:    chID,
		name:    ns,
		trigger: dummy.triggerEvent,
		checkSent: func(message.ID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(umSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

// Tests that the message is rejected when the sized broadcast is malformed.
func Test_userListener_Listen_BadSizedBroadcast(t *testing.T) {
	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(42))
	pub, priv, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	cm := &ChannelMessage{
		Lease:       int64(time.Hour),
		RoundID:     69, // Make the round not match
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	sig := ed25519.Sign(priv, cmSerial)
	ns := &mockNameService{validChMsg: true}

	um := &UserMessage{
		Message:      cmSerial,
		Signature:    sig,
		ECCPublicKey: pub,
	}

	umSerial, err := proto.Marshal(um)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	// Remove half the sized broadcast to make it malformed
	umSerial = umSerial[:len(umSerial)/2]

	// Build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:    chID,
		name:    ns,
		trigger: dummy.triggerEvent,
		checkSent: func(message.ID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(umSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}
