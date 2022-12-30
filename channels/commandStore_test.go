////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/comms/mixmessages"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests that NewCommandStore returns the expected CommandStore.
func TestNewCommandStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	expected := &CommandStore{kv.Prefix(commandStorePrefix)}

	cs := NewCommandStore(kv)

	if !reflect.DeepEqual(expected, cs) {
		t.Errorf("New CommandStore does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, cs)
	}
}

// Tests that a number of channel messages can be saved and loaded from storage.
func TestCommandStore_SaveCommand_LoadCommand(t *testing.T) {
	prng := rand.New(rand.NewSource(430_956))
	cs := NewCommandStore(versioned.NewKV(ekv.MakeMemstore()))

	expected := make([]*CommandMessage, 20)
	for i := range expected {
		nid1 := id.NewIdFromUInt(uint64(i), id.Node, t)
		now := uint64(netTime.Now().UnixNano())
		ri := &mixmessages.RoundInfo{
			ID:        prng.Uint64(),
			UpdateID:  prng.Uint64(),
			State:     prng.Uint32(),
			BatchSize: prng.Uint32(),
			Topology:  [][]byte{nid1.Bytes()},
			Timestamps: []uint64{now - 1000, now - 800, now - 600, now - 400,
				now - 200, now, now + 200},
			Errors: []*mixmessages.RoundError{{
				Id:     prng.Uint64(),
				NodeId: nid1.Bytes(),
				Error:  "Test error",
			}},
			ResourceQueueTimeoutMillis: prng.Uint32(),
			AddressSpaceSize:           prng.Uint32(),
		}
		e := &CommandMessage{
			ChannelID:            randChannelID(prng, t),
			MessageID:            randMessageID(prng, t),
			MessageType:          randAction(prng),
			Nickname:             "George",
			Content:              randPayload(prng, t),
			EncryptedPayload:     randPayload(prng, t),
			PubKey:               randPayload(prng, t),
			Codeset:              uint8(prng.Uint32()),
			Timestamp:            randTimestamp(prng),
			OriginatingTimestamp: randTimestamp(prng),
			Lease:                randLease(prng),
			OriginatingRound:     id.Round(i),
			Round:                rounds.MakeRound(ri),
			Status:               SentStatus(prng.Uint32()),
			FromAdmin:            prng.Int()%2 == 0,
			UserMuted:            prng.Int()%2 == 0,
		}
		expected[i] = e

		err := cs.SaveCommand(e.ChannelID, e.MessageID, e.MessageType,
			e.Nickname, e.Content, e.EncryptedPayload, e.PubKey, e.Codeset,
			e.Timestamp, e.OriginatingTimestamp, e.Lease, e.OriginatingRound, e.Round,
			e.Status, e.FromAdmin, e.UserMuted)
		if err != nil {
			t.Errorf("Failed to save message %d: %+v", i, err)
		}
	}

	for i, e := range expected {
		m, err := cs.LoadCommand(e.ChannelID, e.MessageType, e.Content)
		if err != nil {
			t.Errorf("Failed to load message %d: %+v", i, err)
		}

		if !reflect.DeepEqual(e, m) {
			t.Errorf("Message %d does not match expected."+
				"\nexpected: %+v\nreceived: %+v", i, e, m)
		}
	}
}

// Tests that when no message exists in storage, CommandStore.LoadCommand
// returns an error that signifies the object does not exist, as verified by
// KV.Exists.
func TestCommandStore_LoadCommand_EmptyStorageError(t *testing.T) {
	cs := NewCommandStore(versioned.NewKV(ekv.MakeMemstore()))

	_, err := cs.LoadCommand(&id.ID{1}, Delete, []byte("content"))
	if cs.kv.Exists(err) {
		t.Errorf("Incorrect error when message does not exist: %+v", err)
	}
}

// Tests that CommandStore.DeleteCommand deletes all the command messages.
func TestCommandStore_DeleteCommand(t *testing.T) {
	prng := rand.New(rand.NewSource(430_956))
	cs := NewCommandStore(versioned.NewKV(ekv.MakeMemstore()))

	expected := make([]*CommandMessage, 20)
	for i := range expected {
		nid1 := id.NewIdFromUInt(uint64(i), id.Node, t)
		now := uint64(netTime.Now().UnixNano())
		ri := &mixmessages.RoundInfo{
			ID:        prng.Uint64(),
			UpdateID:  prng.Uint64(),
			State:     prng.Uint32(),
			BatchSize: prng.Uint32(),
			Topology:  [][]byte{nid1.Bytes()},
			Timestamps: []uint64{now - 1000, now - 800, now - 600, now - 400,
				now - 200, now, now + 200},
			Errors: []*mixmessages.RoundError{{
				Id:     prng.Uint64(),
				NodeId: nid1.Bytes(),
				Error:  "Test error",
			}},
			ResourceQueueTimeoutMillis: prng.Uint32(),
			AddressSpaceSize:           prng.Uint32(),
		}
		e := &CommandMessage{
			ChannelID:            randChannelID(prng, t),
			MessageID:            randMessageID(prng, t),
			MessageType:          randAction(prng),
			Nickname:             "George",
			Content:              randPayload(prng, t),
			EncryptedPayload:     randPayload(prng, t),
			PubKey:               randPayload(prng, t),
			Codeset:              uint8(prng.Uint32()),
			Timestamp:            randTimestamp(prng),
			OriginatingTimestamp: randTimestamp(prng),
			Lease:                randLease(prng),
			OriginatingRound:     id.Round(i),
			Round:                rounds.MakeRound(ri),
			Status:               SentStatus(prng.Uint32()),
			FromAdmin:            prng.Int()%2 == 0,
			UserMuted:            prng.Int()%2 == 0,
		}
		expected[i] = e

		err := cs.SaveCommand(e.ChannelID, e.MessageID, e.MessageType,
			e.Nickname, e.Content, e.EncryptedPayload, e.PubKey, e.Codeset,
			e.Timestamp, e.OriginatingTimestamp, e.Lease, e.OriginatingRound, e.Round,
			e.Status, e.FromAdmin, e.UserMuted)
		if err != nil {
			t.Errorf("Failed to save message %d: %+v", i, err)
		}
	}

	for i, e := range expected {
		err := cs.DeleteCommand(e.ChannelID, e.MessageType, e.Content)
		if err != nil {
			t.Errorf("Failed to delete message %d: %+v", i, err)
		}
	}

	for i, e := range expected {
		_, err := cs.LoadCommand(e.ChannelID, e.MessageType, e.Content)
		if cs.kv.Exists(err) {
			t.Errorf(
				"Loaded message %d that should have been deleted: %+v", i, err)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Storage Message                                                            //
////////////////////////////////////////////////////////////////////////////////

// Tests that a CommandMessage with a CommandMessage object can be JSON
// marshalled and unmarshalled and that the result matches the original.
func TestCommandMessage_JsonMarshalUnmarshal(t *testing.T) {
	nid1 := id.NewIdFromString("test01", id.Node, t)
	now := uint64(netTime.Now().UnixNano())
	ri := &mixmessages.RoundInfo{
		ID:        5,
		UpdateID:  1,
		State:     2,
		BatchSize: 150,
		Topology:  [][]byte{nid1.Bytes()},
		Timestamps: []uint64{now - 1000, now - 800, now - 600, now - 400,
			now - 200, now, now + 200},
		Errors: []*mixmessages.RoundError{{
			Id:     uint64(49),
			NodeId: nid1.Bytes(),
			Error:  "Test error",
		}},
		ResourceQueueTimeoutMillis: 0,
		AddressSpaceSize:           8,
	}

	m := CommandMessage{
		ChannelID:            id.NewIdFromString("channelID", id.User, t),
		MessageID:            cryptoChannel.MessageID{1, 2, 3},
		MessageType:          Reaction,
		Nickname:             "Nickname",
		Content:              []byte("content"),
		EncryptedPayload:     []byte("EncryptedPayload"),
		PubKey:               []byte("PubKey"),
		Codeset:              12,
		Timestamp:            netTime.Now().UTC().Round(0),
		OriginatingTimestamp: netTime.Now().UTC().Round(0),
		Lease:                56*time.Second + 6*time.Minute + 12*time.Hour,
		Round:                rounds.MakeRound(ri),
		Status:               Delivered,
		FromAdmin:            true,
		UserMuted:            true,
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Failed to JSON marshal CommandMessage: %+v", err)
	}

	var newMessage CommandMessage
	err = json.Unmarshal(data, &newMessage)
	if err != nil {
		t.Fatalf("Failed to JSON unmarshal CommandMessage: %+v", err)
	}

	if !reflect.DeepEqual(m, newMessage) {
		t.Errorf("JSON marshalled and unmarshalled CommandMessage does not "+
			"match original.\nexpected: %+v\nreceived: %+v", m, newMessage)
	}
}

// Tests that a CommandMessage, with all of the fields set to nil, can be JSON
// marshalled and unmarshalled and that the result matches the original.
func TestMessage_JsonMarshalUnmarshal_NilFields(t *testing.T) {
	var m CommandMessage

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("Failed to JSON marshal empty CommandMessage: %+v", err)
	}

	var newMessage CommandMessage
	err = json.Unmarshal(data, &newMessage)
	if err != nil {
		t.Fatalf("Failed to JSON unmarshal empty CommandMessage: %+v", err)
	}

	if !reflect.DeepEqual(m, newMessage) {
		t.Errorf("JSON marshalled and unmarshalled CommandMessage does not "+
			"match original.\nexpected: %+v\nreceived: %+v", m, newMessage)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Fingerprint                                                                //
////////////////////////////////////////////////////////////////////////////////

// Consistency test of newCommandFingerprint.
func Test_newCommandFingerprint_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(420))
	expectedFingerprints := []string{
		"1cfa6553e086f4ff3bd923916c8e01785c608e4b813d4945de048d9b7ef55377",
		"3d4e3329e5b2a87c2b14c6cf50c4fb04c215c00905f2f3c5b010786cb7fe02bc",
		"f8ea81c157cec11d6dada71b7dffd3c6b953f0021c3b6243f80677a66c8482f4",
		"ee155843572f0a8bb43b8b452dc8a96b62176520db440b3eb0f86b95316217ae",
		"c7375d20831a11987dab8ed80edeee99366d7c53a5e93f9dce0cdf8699de101e",
		"2ecd9a78fa220fb9187899b38dbe42292e4a3584abd8b6c79d6ffb513be41a1f",
		"af4a6a01a72239d593a563a2ad5d31bf4eee67c7c536637e17423a85b4113191",
		"7c397a8dfea5fe0dbe80e64fcff2dea5dc654c8c0a99e10468d5b9817af1710d",
		"9d2d9bbb7e1d0bab5f285cfa9d9bbfc3d39103e6dc6dfa30daaa26321e7ed8d2",
		"43c5a17c8b9c6787cd49f5e37d04fa1d19197d5e8732ead280ed8153dd7b7f81",
		"9d48022a31e70045f4e92d06a1c6f923f1f600358c7923ca8a2e0f343f478e6e",
		"cc9142dc4dd26617cfc5263fb31ce2446d695f9a69fe0edb6bdfe73fa9131725",
		"bbc8cfbc47a46cf101519c9537dadad81a91be395f1e9750c2ebb97591e0ed4f",
		"3d611fe8bf723e378c97fc4fd1f23ad85cc208b424953dbc5d64d81c38bed455",
		"9ed9cb3ae4a18c16367f77253f701da79b6fecf1c9c5cb3b6e27ab9ea9d76b7f",
		"0f534845bdc574424a0b72a1f382c30b9c2d5302025117062dec8b5c5fdceafc",
	}

	for i, expected := range expectedFingerprints {
		fp := newCommandFingerprint(randChannelID(prng, t),
			randAction(prng), randPayload(prng, t))

		if expected != fp.String() {
			t.Errorf("leaseFingerprint does not match expected (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, fp)
		}
	}
}

// Tests that any changes to any of the inputs to newCommandFingerprint result
// in different fingerprints.
func Test_newCommandFingerprint_Uniqueness(t *testing.T) {
	rng := csprng.NewSystemRNG()
	const n = 100
	chanIDs := make([]*id.ID, n)
	payloads, encryptedPayloads := make([][]byte, n), make([][]byte, n)
	for i := 0; i < n; i++ {
		chanIDs[i] = randChannelID(rng, t)
		payloads[i] = randPayload(rng, t)
		encryptedPayloads[i] = randPayload(rng, t)

	}
	commands := []MessageType{Delete, Pinned, Mute}

	fingerprints := make(map[string]bool)
	for _, channelID := range chanIDs {
		for _, payload := range payloads {
			for _, command := range commands {
				fp := newCommandFingerprint(channelID, command, payload)
				if fingerprints[fp.String()] {
					t.Errorf("Fingerprint %s already exists.", fp)
				}

				fingerprints[fp.String()] = true
			}
		}
	}
}
