////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"gitlab.com/elixxir/client/broadcast"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

type triggerAdminEventDummy struct {
	gotData bool

	chID        *id.ID
	cm          *ChannelMessage
	msgID       cryptoChannel.MessageID
	receptionID receptionID.EphemeralIdentity
	round       rounds.Round
}

func (taed *triggerAdminEventDummy) triggerAdminEvent(chID *id.ID,
	cm *ChannelMessage, messageID cryptoChannel.MessageID,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	taed.gotData = true

	taed.chID = chID
	taed.cm = cm
	taed.msgID = messageID
	taed.receptionID = receptionID
	taed.round = round
}

// Tests the happy path.
func TestAdminListener_Listen(t *testing.T) {

	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = time.Now()

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

	chMsgSerialSized, err := broadcast.NewSizedBroadcast(
		512, cmSerial)
	if err != nil {
		t.Fatalf("Failed to size channel message: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(cmSerial)

	// Build the listener
	dummy := &triggerAdminEventDummy{}

	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
	}

	// Call the listener
	al.Listen(chMsgSerialSized, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if !dummy.gotData {
		t.Fatalf("No data returned after valid listen")
	}

	if !dummy.chID.Cmp(chID) {
		t.Errorf("Channel ID not correct: %s vs %s", dummy.chID, chID)
	}

	if !bytes.Equal(cm.Payload, dummy.cm.Payload) {
		t.Errorf("payload not correct: %s vs %s", cm.Payload,
			dummy.cm.Payload)
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

// Tests that the message is rejected when the round it came on doesn't match
// the round in the channel message.
func TestAdminListener_Listen_BadRound(t *testing.T) {

	// build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = time.Now()

	cm := &ChannelMessage{
		Lease: int64(time.Hour),
		// Different from the round above
		RoundID:     69,
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	chMsgSerialSized, err := broadcast.NewSizedBroadcast(
		512, cmSerial)
	if err != nil {
		t.Fatalf("Failed to size channel message: %+v", err)
	}

	// Build the listener
	dummy := &triggerAdminEventDummy{}

	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
	}

	// Call the listener
	al.Listen(chMsgSerialSized, receptionID.EphemeralIdentity{}, r)

	// check the results
	if dummy.gotData {
		t.Fatalf("payload handled when it should have failed due to " +
			"a round issue")
	}

}

// Tests that the message is rejected when the channel message is malformed.
func TestAdminListener_Listen_BadChannelMessage(t *testing.T) {

	// Build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = time.Now()

	cmSerial := []byte("blarg")

	chMsgSerialSized, err := broadcast.NewSizedBroadcast(
		512, cmSerial)
	if err != nil {
		t.Fatalf("Failed to size channel message: %+v", err)
	}

	// Build the listener
	dummy := &triggerAdminEventDummy{}

	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
	}

	// Call the listener
	al.Listen(chMsgSerialSized, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("payload handled when it should have failed due to " +
			"a malformed channel message")
	}

}

// Tests that the message is rejected when the sized broadcast message is
// malformed.
func TestAdminListener_Listen_BadSizedBroadcast(t *testing.T) {

	// build inputs
	chID := &id.ID{}
	chID[0] = 1

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = time.Now()

	cm := &ChannelMessage{
		Lease: int64(time.Hour),
		// Different from the round above
		RoundID:     69,
		PayloadType: 42,
		Payload:     []byte("blarg"),
	}

	cmSerial, err := proto.Marshal(cm)
	if err != nil {
		t.Fatalf("Failed to marshal proto: %+v", err)
	}

	chMsgSerialSized, err := broadcast.NewSizedBroadcast(
		512, cmSerial)
	if err != nil {
		t.Fatalf("Failed to size channel message: %+v", err)
	}

	// Remove half the sized broadcast to make it malformed
	chMsgSerialSized = chMsgSerialSized[:broadcast.GetSizedBroadcastSize(chMsgSerialSized)/2]

	// Build the listener
	dummy := &triggerAdminEventDummy{}

	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
	}

	// Call the listener
	al.Listen(chMsgSerialSized, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("payload handled when it should have failed due to " +
			"a malformed sized broadcast")
	}
}
