////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

type triggerAdminEventDummy struct {
	gotData bool

	chID             *id.ID
	cm               *ChannelMessage
	encryptedPayload []byte
	msgID            cryptoChannel.MessageID
	receptionID      receptionID.EphemeralIdentity
	round            rounds.Round
}

func (taed *triggerAdminEventDummy) triggerAdminEvent(chID *id.ID,
	cm *ChannelMessage, encryptedPayload []byte, _ time.Time,
	messageID cryptoChannel.MessageID,
	receptionID receptionID.EphemeralIdentity, round rounds.Round,
	_ SentStatus) (uint64, error) {
	taed.gotData = true

	taed.chID = chID
	taed.cm = cm
	taed.encryptedPayload = encryptedPayload
	taed.msgID = messageID
	taed.receptionID = receptionID
	taed.round = round

	return 0, nil
}

// Tests the happy path.
func Test_adminListener_Listen(t *testing.T) {
	// Build inputs
	chID := &id.ID{1}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
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
	msgID := cryptoChannel.MakeMessageID(cmSerial, chID)

	// Build the listener
	dummy := &triggerAdminEventDummy{}
	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
		checkSent: func(cryptoChannel.MessageID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(cmSerial, nil, receptionID.EphemeralIdentity{}, r)

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

// Tests that the message is rejected when the channel message is malformed.
func TestAdminListener_Listen_BadChannelMessage(t *testing.T) {

	// Build inputs
	chID := &id.ID{1}

	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}

	cmSerial := []byte("blarg")

	// Build the listener
	dummy := &triggerAdminEventDummy{}

	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
		checkSent: func(cryptoChannel.MessageID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(cmSerial, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("payload handled when it should have failed due to " +
			"a malformed channel message")
	}
}

// Tests that the message is rejected when the sized broadcast message is
// malformed.
func TestAdminListener_Listen_BadSizedBroadcast(t *testing.T) {
	// Build inputs
	chID := &id.ID{1}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	chMsgSerialSized := []byte("invalid")

	// Build the listener
	dummy := &triggerAdminEventDummy{}
	al := adminListener{
		chID:    chID,
		trigger: dummy.triggerAdminEvent,
		checkSent: func(cryptoChannel.MessageID, rounds.Round) bool {
			return false
		},
	}

	// Call the listener
	al.Listen(chMsgSerialSized, nil, receptionID.EphemeralIdentity{}, r)

	// Check the results
	if dummy.gotData {
		t.Fatalf("payload handled when it should have failed due to " +
			"a malformed sized broadcast")
	}
}
