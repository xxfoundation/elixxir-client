///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
)

// Tests happy path of LoadOrMakeConversation when making a new Conversation.
func TestLoadOrMakeConversation_New(t *testing.T) {
	// Set up test values
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := &Conversation{
		lastReceivedID:         0,
		numReceivedRevolutions: 0,
		nextSentID:             0,
		partner:                partner,
		kv:                     kv,
	}

	// Create new Conversation
	conv := LoadOrMakeConversation(kv, partner)

	// Check that the result matches the expected Conversation
	if !reflect.DeepEqual(expectedConv, conv) {
		t.Errorf("LoadOrMakeConversation made unexpected Conversation."+
			"\nexpected: %+v\nreceived: %+v", expectedConv, conv)
	}
}

// Tests happy path of LoadOrMakeConversation when loading a Conversation.
func TestLoadOrMakeConversation_Load(t *testing.T) {
	// Set up test values
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := LoadOrMakeConversation(kv, partner)

	// Load Conversation
	conv := LoadOrMakeConversation(kv, partner)

	// Check that the result matches the expected Conversation
	if !reflect.DeepEqual(expectedConv, conv) {
		t.Errorf("LoadOrMakeConversation made unexpected Conversation."+
			"\nexpected: %+v\nreceived: %+v", expectedConv, conv)
	}
}

// Tests case 1 of Conversation.ProcessReceivedMessageID.
func TestConversation_ProcessReceivedMessageID_Case_1(t *testing.T) {
	// Set up test values
	mid := uint32(5)
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := LoadOrMakeConversation(kv, partner)
	expectedConv.lastReceivedID = mid
	expectedConv.numReceivedRevolutions = 1
	conv := LoadOrMakeConversation(kv, partner)
	conv.lastReceivedID = topRegion + 5
	expectedResult := uint64(expectedConv.numReceivedRevolutions)<<32 | uint64(mid)

	result := conv.ProcessReceivedMessageID(mid)
	if result != expectedResult {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"result.\nexpected: %+v\n\trecieved: %+v", expectedResult, result)
	}
	if !reflect.DeepEqual(expectedConv, conv) {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"Conversation.\nexpected: %+v\n\trecieved: %+v", expectedConv, conv)
	}
}

// Tests case 0 of Conversation.ProcessReceivedMessageID.
func TestConversation_ProcessReceivedMessageID_Case_0(t *testing.T) {
	// Set up test values
	mid := uint32(5)
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := LoadOrMakeConversation(kv, partner)
	expectedConv.lastReceivedID = mid
	conv := LoadOrMakeConversation(kv, partner)
	expectedResult := uint64(expectedConv.numReceivedRevolutions)<<32 | uint64(mid)

	result := conv.ProcessReceivedMessageID(mid)
	if result != expectedResult {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"result.\nexpected: %+v\n\trecieved: %+v", expectedResult, result)
	}
	if !reflect.DeepEqual(expectedConv, conv) {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"Conversation.\nexpected: %+v\n\trecieved: %+v", expectedConv, conv)
	}
}

// Tests case -1 of Conversation.ProcessReceivedMessageID.
func TestConversation_ProcessReceivedMessageID_Case_Neg1(t *testing.T) {
	// Set up test values
	mid := uint32(topRegion + 5)
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := LoadOrMakeConversation(kv, partner)
	expectedConv.lastReceivedID = bottomRegion - 5
	conv := LoadOrMakeConversation(kv, partner)
	conv.lastReceivedID = bottomRegion - 5
	expectedResult := uint64(expectedConv.numReceivedRevolutions-1)<<32 | uint64(mid)

	result := conv.ProcessReceivedMessageID(mid)
	if result != expectedResult {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"result.\nexpected: %+v\n\trecieved: %+v", expectedResult, result)
	}
	if !reflect.DeepEqual(expectedConv, conv) {
		t.Errorf("ProcessReceivedMessageID did not product the expected "+
			"Conversation.\nexpected: %+v\n\trecieved: %+v", expectedConv, conv)
	}
}

// Tests happy path of Conversation.GetNextSendID.
func TestConversation_GetNextSendID(t *testing.T) {
	// Set up test values
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	conv := LoadOrMakeConversation(kv, partner)
	conv.nextSentID = maxTruncatedID - 100

	for i := uint64(maxTruncatedID - 100); i < maxTruncatedID+100; i++ {
		fullID, truncID := conv.GetNextSendID()
		if fullID != i {
			t.Errorf("Returned incorrect full sendID."+
				"\nexpected: %d\nreceived: %d", i, fullID)
		}
		if truncID != uint32(i) {
			t.Errorf("Returned incorrect truncated sendID."+
				"\nexpected: %d\nreceived: %d", uint32(i), truncID)
		}
	}
}

// Tests the happy path of save and loadConversation.
func TestConversation_save_load(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	expectedConv := makeRandomConv(kv, partner)
	expectedErr := "loadConversation produced an error: Failed to Load " +
		"conversation: object not found"

	err := expectedConv.save()
	if err != nil {
		t.Errorf("save produced an error: %v", err)
	}

	testConv, err := loadConversation(kv, partner)
	if err != nil {
		t.Errorf("loadConversation produced an error: %v", err)
	}

	if !reflect.DeepEqual(expectedConv, testConv) {
		t.Errorf("saving and loading Conversation failed."+
			"\nexpected: %+v\nreceived: %+v", expectedConv, testConv)
	}

	_, err = loadConversation(versioned.NewKV(ekv.MakeMemstore()), partner)
	if err == nil {
		t.Errorf("loadConversation failed to produce an error."+
			"\nexpected: %s\nreceived: %v", expectedErr, nil)
	}
}

// Happy path.
func TestConversation_Delete(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	partner := id.NewIdFromString("partner ID", id.User, t)
	conv := makeRandomConv(kv, partner)

	if err := conv.save(); err != nil {
		t.Fatalf("Failed to save conversation to storage: %+v", err)
	}

	if _, err := loadConversation(kv, partner); err != nil {
		t.Fatalf("Failed to load conversation from storage: %v", err)
	}

	if err := conv.delete(); err != nil {
		t.Errorf("delete produced an error: %+v", err)
	}

	if _, err := loadConversation(kv, partner); err == nil {
		t.Error("Object found in storage when it should be deleted.")
	}
}

// Tests the happy path of marshal and unmarshal.
func TestConversation_marshal_unmarshal(t *testing.T) {
	expectedConv := makeRandomConv(versioned.NewKV(ekv.MakeMemstore()),
		id.NewIdFromString("partner ID", id.User, t))
	testConv := LoadOrMakeConversation(expectedConv.kv, expectedConv.partner)

	data, err := expectedConv.marshal()
	if err != nil {
		t.Errorf("marshal returned an error: %v", err)
	}

	err = testConv.unmarshal(data)
	if err != nil {
		t.Errorf("unmarshal returned an error: %v", err)
	}

	if !reflect.DeepEqual(expectedConv, testConv) {
		t.Errorf("marshaling and unmarshaling Conversation failed."+
			"\nexpected: %+v\nreceived: %+v", expectedConv, testConv)
	}
}

func makeRandomConv(kv *versioned.KV, partner *id.ID) *Conversation {
	c := LoadOrMakeConversation(kv, partner)
	c.lastReceivedID = rand.Uint32()
	c.numReceivedRevolutions = rand.Uint32()
	c.nextSentID = rand.Uint64()

	return c
}
