////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"bytes"
	sidhinterface "gitlab.com/elixxir/client/v4/auth/store"
	"math/rand"
	"reflect"
	"testing"

	"gitlab.com/xx_network/primitives/id"
)

// Tests newBaseFormat
func TestNewBaseFormat(t *testing.T) {
	// Construct message
	pubKeySize := 256
	payloadSize := pubKeySize + sidhinterface.PubKeyByteSize + 2
	baseMsg := newBaseFormat(payloadSize, pubKeySize)

	if baseMsg.GetVersion() != requestFmtVersion {
		t.Errorf("Incorrect version: %d, expect %d",
			baseMsg.GetVersion(), requestFmtVersion)
	}

	// Check that the base format was constructed properly
	if !bytes.Equal(baseMsg.pubkey, make([]byte, pubKeySize)) {
		t.Errorf("NewBaseFormat error: "+
			"Unexpected pubkey field in base format."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", make([]byte, pubKeySize), baseMsg.pubkey)
	}

	expectedEcrPayloadSize := payloadSize - (pubKeySize) - 1
	if !bytes.Equal(baseMsg.ecrPayload, make([]byte, expectedEcrPayloadSize)) {
		t.Errorf("NewBaseFormat error: "+
			"Unexpected payload field in base format."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", make([]byte, expectedEcrPayloadSize), baseMsg.ecrPayload)
	}

	// Error case, where payload size is less than the public key
	defer func() {
		if r := recover(); r == nil {
			t.Error("newBaseFormat() did not panic when the size of " +
				"the payload is smaller than the size of the public key.")
		}
	}()

	newBaseFormat(0, pubKeySize)
}

/* Tests the setter/getter methods for baseFormat */

// Set/get PubKey tests
func TestBaseFormat_SetGetPubKey(t *testing.T) {
	// Construct message
	pubKeySize := 256
	payloadSize := pubKeySize + sidhinterface.PubKeyByteSize + 2
	baseMsg := newBaseFormat(payloadSize, pubKeySize)

	// Test setter
	grp := getGroup()
	pubKey := grp.NewInt(25)
	baseMsg.SetPubKey(pubKey)
	expectedBytes := pubKey.LeftpadBytes(uint64(len(baseMsg.pubkey)))
	if !bytes.Equal(baseMsg.pubkey, expectedBytes) {
		t.Errorf("SetPubKey() error: "+
			"Public key field does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", expectedBytes, baseMsg.pubkey)
	}

	// Test getter
	receivedKey := baseMsg.GetPubKey(grp)
	if !bytes.Equal(pubKey.Bytes(), receivedKey.Bytes()) {
		t.Errorf("GetPubKey() error: "+
			"Public key retrieved does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", pubKey, receivedKey)
	}

}

// Set/get EcrPayload tests
func TestBaseFormat_SetGetEcrPayload(t *testing.T) {
	// Construct message
	pubKeySize := 256
	payloadSize := (pubKeySize + sidhinterface.PubKeyByteSize) * 2
	baseMsg := newBaseFormat(payloadSize, pubKeySize)

	// Test setter
	ecrPayloadSize := payloadSize - (pubKeySize) - 1
	ecrPayload := newPayload(ecrPayloadSize, "ecrPayload")
	baseMsg.SetEcrPayload(ecrPayload)
	if !bytes.Equal(ecrPayload, baseMsg.ecrPayload) {
		t.Errorf("SetEcrPayload() error: "+
			"EcrPayload field does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", ecrPayload, baseMsg.ecrPayload)

	}

	// Test Getter
	receivedEcrPayload := baseMsg.GetEcrPayload()
	if !bytes.Equal(receivedEcrPayload, ecrPayload) {
		t.Errorf("GetEcrPayload() error: "+
			"EcrPayload retrieved does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", ecrPayload, receivedEcrPayload)
	}

	// Setter error path: Setting ecrPayload that
	// does not completely fill field
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetEcrPayload() did not panic when the size of " +
				"the ecrPayload is smaller than the pre-constructed field.")
		}
	}()
	baseMsg.SetEcrPayload([]byte("ecrPayload"))
}

// Marshal/ unmarshal tests
func TestBaseFormat_MarshalUnmarshal(t *testing.T) {
	// Construct a fully populated message
	pubKeySize := 256
	payloadSize := (pubKeySize + sidhinterface.PubKeyByteSize) * 2
	baseMsg := newBaseFormat(payloadSize, pubKeySize)
	ecrPayloadSize := payloadSize - (pubKeySize) - 1
	ecrPayload := newPayload(ecrPayloadSize, "ecrPayload")
	baseMsg.SetEcrPayload(ecrPayload)
	grp := getGroup()
	pubKey := grp.NewInt(25)
	baseMsg.SetPubKey(pubKey)

	// Test marshal
	data := baseMsg.Marshal()
	if !bytes.Equal(data, baseMsg.data) {
		t.Errorf("baseFormat.Marshal() error: "+
			"Marshalled data is not expected."+
			"\n\tExpected: %v\n\tReceived: %v", baseMsg.data, data)
	}

	// Test unmarshal
	newMsg, err := unmarshalBaseFormat(data, pubKeySize)
	if err != nil {
		t.Errorf("unmarshalBaseFormat() error: "+
			"Could not unmarshal into baseFormat: %v", err)
	}

	if !reflect.DeepEqual(*newMsg, baseMsg) {
		t.Errorf("unmarshalBaseFormat() error: "+
			"Unmarshalled message does not match originally marshalled message."+
			"\n\tExpected: %v\n\tRecieved: %v", baseMsg, *newMsg)
	}

	// Unmarshal error test: Invalid size parameter
	_, err = unmarshalBaseFormat(make([]byte, 0), pubKeySize)
	if err == nil {
		t.Errorf("unmarshalBaseFormat() error: " +
			"Should not be able to unmarshal when baseFormat is too small")
	}

}

// Tests newEcrFormat
func TestNewEcrFormat(t *testing.T) {
	// Construct message
	payloadSize := ownershipSize*2 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)

	// Check that the ecrFormat was constructed properly
	if !bytes.Equal(ecrMsg.ownership, make([]byte, ownershipSize)) {
		t.Errorf("newEcrFormat error: "+
			"Unexpected ownership field in ecrFormat."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", make([]byte, payloadSize), ecrMsg.ownership)
	}

	if !bytes.Equal(ecrMsg.payload, make([]byte,
		payloadSize-ownershipSize-sidhinterface.PubKeyByteSize-1)) {
		t.Errorf("newEcrFormat error: "+
			"Unexpected ownership field in ecrFormat."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", make([]byte, payloadSize-ownershipSize), ecrMsg.payload)
	}

	// Error case, where payload size is less than the public key
	defer func() {
		if r := recover(); r == nil {
			t.Error("newEcrFormat() did not panic when the size of " +
				"the payload is smaller than the size of the ownership")
		}
	}()

	newEcrFormat(0)
}

/* Tests the setter/getter methods for ecrFormat */

// Set/get ownership tests
func TestEcrFormat_SetGetOwnership(t *testing.T) {
	// Construct message
	payloadSize := ownershipSize*2 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)

	// Test setter
	ownership := newOwnership("owner")
	ecrMsg.SetOwnership(ownership)
	if !bytes.Equal(ownership, ecrMsg.ownership) {
		t.Errorf("SetOwnership() error: "+
			"Ownership field does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", ownership, ecrMsg.ownership)

	}

	// Test getter
	receivedOwnership := ecrMsg.GetOwnership()
	if !bytes.Equal(receivedOwnership, ecrMsg.ownership) {
		t.Errorf("GetOwnership() error: "+
			"Ownership retrieved does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", ownership, receivedOwnership)

	}

	// Test setter error path: Setting ownership of incorrect size
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetOwnership() did not panic when the size of " +
				"the ownership is smaller than the required ownership size.")
		}
	}()

	ecrMsg.SetOwnership([]byte("ownership"))
}

// Set/get payload tests
func TestEcrFormat_SetGetPayload(t *testing.T) {
	// Construct message
	payloadSize := ownershipSize*2 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)

	// Test set
	expectedPayload := newPayload(
		payloadSize-ownershipSize-sidhinterface.PubKeyByteSize-1,
		"ownership")
	ecrMsg.SetPayload(expectedPayload)

	if !bytes.Equal(expectedPayload, ecrMsg.payload) {
		t.Errorf("SetPayload() error: "+
			"Payload field does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", expectedPayload, ecrMsg.payload)
	}

	// Test get
	receivedPayload := ecrMsg.GetPayload()
	if !bytes.Equal(receivedPayload, expectedPayload) {
		t.Errorf("GetPayload() error: "+
			"Payload retrieved does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", expectedPayload, receivedPayload)

	}

	// Test setter error path: Setting payload of incorrect size
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetPayload() did not panic when the size of " +
				"the payload is smaller than the required payload size.")
		}
	}()

	ecrMsg.SetPayload([]byte("payload"))
}

// Marshal/ unmarshal tests
func TestEcrFormat_MarshalUnmarshal(t *testing.T) {
	// Construct message
	payloadSize := ownershipSize*2 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)
	expectedPayload := newPayload(
		payloadSize-ownershipSize-sidhinterface.PubKeyByteSize-1,
		"ownership")
	ecrMsg.SetPayload(expectedPayload)
	ownership := newOwnership("owner")
	ecrMsg.SetOwnership(ownership)

	// Test marshal
	data := ecrMsg.Marshal()
	if !bytes.Equal(data, ecrMsg.data) {
		t.Errorf("ecrFormat.Marshal() error: "+
			"Marshalled data is not expected."+
			"\n\tExpected: %v\n\tReceived: %v", ecrMsg.data, data)
	}

	// Test unmarshal
	newMsg, err := unmarshalEcrFormat(data)
	if err != nil {
		t.Errorf("unmarshalEcrFormat() error: "+
			"Could not unmarshal into ecrFormat: %v", err)
	}

	if !reflect.DeepEqual(newMsg, ecrMsg) {
		t.Errorf("unmarshalBaseFormat() error: "+
			"Unmarshalled message does not match originally marshalled message."+
			"\n\tExpected: %v\n\tRecieved: %v", ecrMsg, newMsg)
	}

	// Unmarshal error test: Invalid size parameter
	_, err = unmarshalEcrFormat(make([]byte, 0))
	if err == nil {
		t.Errorf("unmarshalEcrFormat() error: " +
			"Should not be able to unmarshal when ecrFormat is too small")
	}

}

// Tests newRequestFormat
func TestNewRequestFormat(t *testing.T) {
	// Construct message
	payloadSize := id.ArrIDLen*2 - 1 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)
	expectedPayload := newPayload(id.ArrIDLen, "ownership")
	ecrMsg.SetPayload(expectedPayload)
	reqMsg, err := newRequestFormat(ecrMsg)
	if err != nil {
		t.Fatalf("newRequestFormat() error: "+
			"Failed to construct message: %v", err)
	}

	// Check that the requestFormat was constructed properly
	if !bytes.Equal(reqMsg.id, expectedPayload) {
		t.Errorf("newRequestFormat() error: "+
			"Unexpected id field in requestFormat."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", make([]byte, id.ArrIDLen), reqMsg.id)
	}

	// FIXME: Commented out for now.. it's not clear why this was necessary
	// if !bytes.Equal(reqMsg.GetPayload(), make([]byte, 0,
	// 	sidhinterface.PubKeyByteSize)) {
	// 	t.Errorf("newRequestFormat() error: "+
	// 		"Unexpected msgPayload field in requestFormat."+
	// 		"\n\tExpected: %v"+
	// 		"\n\tReceived: %v", make([]byte, 0), reqMsg.GetPayload())
	// }

	payloadSize = ownershipSize*2 + sidhinterface.PubKeyByteSize + 1
	ecrMsg = newEcrFormat(payloadSize)
	reqMsg, err = newRequestFormat(ecrMsg)
	if err == nil {
		t.Errorf("Expecter error: Should be invalid size when calling newRequestFormat")
	}

}

/* Setter/Getter tests for RequestFormat */

// Unit test for get/SetID
func TestRequestFormat_SetGetID(t *testing.T) {
	// Construct message
	payloadSize := id.ArrIDLen*2 - 1 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)
	expectedPayload := newPayload(id.ArrIDLen, "ownership")
	ecrMsg.SetPayload(expectedPayload)
	reqMsg, err := newRequestFormat(ecrMsg)
	if err != nil {
		t.Fatalf("newRequestFormat() error: "+
			"Failed to construct message: %v", err)
	}

	// Test SetID
	prng := rand.New(rand.NewSource(42))
	expectedId := randID(prng, id.User)
	reqMsg.SetID(expectedId)
	if !bytes.Equal(reqMsg.id, expectedId.Bytes()) {
		t.Errorf("SetID() error: "+
			"Id field does not have expected value."+
			"\n\tExpected: %v\n\tReceived: %v", expectedId, reqMsg.GetPayload())
	}

	// Test GetID
	receivedId, err := reqMsg.GetID()
	if err != nil {
		t.Fatalf("GetID() error: "+
			"Retrieved id does not match expected value:"+
			"\n\tExpected: %v\n\tReceived: %v", expectedId, receivedId)
	}

	// Test GetID error: unmarshal-able ID in requestFormat
	reqMsg.id = []byte("badId")
	receivedId, err = reqMsg.GetID()
	if err == nil {
		t.Errorf("GetID() error: " +
			"Should not be able get ID from request message ")
	}

}

// Unit test for get/SetMsgPayload
func TestRequestFormat_SetGetMsgPayload(t *testing.T) {
	// Construct message
	payloadSize := id.ArrIDLen*3 - 1 + sidhinterface.PubKeyByteSize + 1
	ecrMsg := newEcrFormat(payloadSize)
	expectedPayload := newPayload(id.ArrIDLen*2, "ownership")
	ecrMsg.SetPayload(expectedPayload)
	reqMsg, err := newRequestFormat(ecrMsg)
	if err != nil {
		t.Fatalf("newRequestFormat() error: "+
			"Failed to construct message: %v", err)
	}

	// Test SetMsgPayload
	msgPayload := newPayload(id.ArrIDLen*2,
		"msgPayload")
	reqMsg.SetPayload(msgPayload)
	if !bytes.Equal(reqMsg.GetPayload(), msgPayload) {
		t.Errorf("SetMsgPayload() error: "+
			"MsgPayload has unexpected value: "+
			"\n\tExpected: %v\n\tReceived: %v", msgPayload, reqMsg.GetPayload())
	}

	// Test GetMsgPayload
	retrievedMsgPayload := reqMsg.GetPayload()
	if !bytes.Equal(retrievedMsgPayload, msgPayload) {
		t.Errorf("GetMsgPayload() error: "+
			"MsgPayload has unexpected value: "+
			"\n\tExpected: %v\n\tReceived: %v", msgPayload, retrievedMsgPayload)

	}

	// Test SetMsgPayload error: Invalid message payload size
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetMsgPayload() did not panic when the size of " +
				"the payload is the incorrect size.")
		}
	}()
	expectedPayload = append(expectedPayload, expectedPayload...)
	reqMsg.SetPayload(expectedPayload)
}
