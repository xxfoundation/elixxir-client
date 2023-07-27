////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/message"
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
	expectedKv, err := kv.Prefix(commandStorePrefix)
	require.NoError(t, err)
	expected := &CommandStore{kv: expectedKv}

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
		MessageID:            message.ID{1, 2, 3},
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
		"6b265ea2e969471671f7d53bbb9a0dda51576be2c69694ab1ecab4b41f622cf1",
		"2d53cec31aacdc1ab0156c6b88324736b213925708ce0945fb8faa16d94902fa",
		"1ee9a721c47319be5d4a8e09d69530a234b7fc7c00115f53e6418c057e951b52",
		"4b9f35d31027b61ae535628cccf4b68f892b8d199a9e8608135b98f6c8a4b689",
		"0d4735c94d17529d24bfffb87f7b04ea63349defe6109bcf2edabbe97ffad0ba",
		"1edb044b963499287baefc985d4d063018bc8258da174b7f691860db8cfaded6",
		"3febcb023837af04b5375e5e013b2df1f26e7062fd7ce208b4a06c784dc3e4d2",
		"58c29332fe7721710e85cb1464cb5781b61df7cee3c4729a375b573b8dcac58f",
		"0473d3c35e64b5180645adec265d9567feb063f4618e5e186774e8b74fbab72c",
		"df40563e8bc3a80b5c16faae2d9525489868ee8fddb01cb55486a15b35dae0fc",
		"8bdba326cf4470b07221ab5c40b0c5158e6966f102e764edbab7a1349c530b5e",
		"0e75c8e4761d0c0934fd252d98e902fb4bf000d969b73a53b3a64befc772c7e0",
		"3888d2fdd038a4ffec86ecde245955c35657b5f80bd80ee0a57783255e655479",
		"ef73248ca1c148c3d294da6438dc6fabcf1ff4f0f8f544527f27d82c4ce57a6a",
		"f81dfa2ecdfec4f7a7fb594ecba289d4cbdf1de50f8ca14af6405b1d26fb64a1",
		"b4865a5687c8a74b7823938dafc7b0de2977756c36439175144811d8108486d5",
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
