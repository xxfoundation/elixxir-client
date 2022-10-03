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
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

type triggerEventDummy struct {
	gotData bool

	chID        *id.ID
	umi         *userMessageInternal
	msgID       cryptoChannel.MessageID
	receptionID receptionID.EphemeralIdentity
	round       rounds.Round
}

func (ted *triggerEventDummy) triggerEvent(chID *id.ID, umi *userMessageInternal,
	ts time.Time, receptionID receptionID.EphemeralIdentity, round rounds.Round,
	sent SentStatus) (uint64, error) {
	ted.gotData = true

	ted.chID = chID
	ted.umi = umi
	ted.receptionID = receptionID
	ted.round = round
	ted.msgID = umi.GetMessageID()

	return 0, nil
}

// Tests the happy path
func TestUserListener_Listen(t *testing.T) {

	//build inputs
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

	msgID := cryptoChannel.MakeMessageID(cmSerial)

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

	//build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:      chID,
		name:      ns,
		trigger:   dummy.triggerEvent,
		checkSent: func(messageID cryptoChannel.MessageID) bool { return false },
	}

	//call the listener
	al.Listen(umSerial, receptionID.EphemeralIdentity{}, r)

	//check the results
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

//tests that the message is rejected when the user signature is invalid
func TestUserListener_Listen_BadUserSig(t *testing.T) {

	//build inputs
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

	_, badpriv, err := ed25519.GenerateKey(rng)
	if err != nil {
		t.Fatalf("failed to generate ed25519 keypair, cant run test")
	}

	sig := ed25519.Sign(badpriv, cmSerial)
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

	//build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:      chID,
		name:      ns,
		trigger:   dummy.triggerEvent,
		checkSent: func(messageID cryptoChannel.MessageID) bool { return false },
	}

	//call the listener
	al.Listen(umSerial, receptionID.EphemeralIdentity{}, r)

	//check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

//tests that the message is rejected when the round in the message does not
//match the round passed in
func TestUserListener_Listen_BadRound(t *testing.T) {

	//build inputs
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
		Lease: int64(time.Hour),
		//make the round not match
		RoundID:     69,
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

	//build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:      chID,
		name:      ns,
		trigger:   dummy.triggerEvent,
		checkSent: func(messageID cryptoChannel.MessageID) bool { return false },
	}

	//call the listener
	al.Listen(umSerial, receptionID.EphemeralIdentity{}, r)

	//check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

//tests that the message is rejected when the user message is malformed
func TestUserListener_Listen_BadMessage(t *testing.T) {

	//build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	ns := &mockNameService{validChMsg: true}

	umSerial := []byte("malformed")

	//build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:      chID,
		name:      ns,
		trigger:   dummy.triggerEvent,
		checkSent: func(messageID cryptoChannel.MessageID) bool { return false },
	}

	//call the listener
	al.Listen(umSerial, receptionID.EphemeralIdentity{}, r)

	//check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}

//tests that the message is rejected when the sized broadcast is malformed
func TestUserListener_Listen_BadSizedBroadcast(t *testing.T) {

	//build inputs
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
		Lease: int64(time.Hour),
		//make the round not match
		RoundID:     69,
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

	//remove half the sized broadcast to make it malformed
	umSerial = umSerial[:len(umSerial)/2]

	//build the listener
	dummy := &triggerEventDummy{}

	al := userListener{
		chID:      chID,
		name:      ns,
		trigger:   dummy.triggerEvent,
		checkSent: func(messageID cryptoChannel.MessageID) bool { return false },
	}

	//call the listener
	al.Listen(umSerial, receptionID.EphemeralIdentity{}, r)

	//check the results
	if dummy.gotData {
		t.Fatalf("Data returned after invalid listen")
	}
}
