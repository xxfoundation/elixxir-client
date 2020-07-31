package keyStore

import (
	"bytes"
	"encoding/gob"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// Test that the buffer is recieving objects and that it is in fact circular
func TestPush(t *testing.T) {
	aBuffer := NewReceptionKeyManagerBuffer()

	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewIdFromUInt(14, id.User, t)
	userID := id.NewIdFromUInt(18, id.User, t)

	//Generate twice the amount of keymanagers so we can test the circularness of the buffer as well
	kmArray := []KeyManager{}
	for i := 0; i < ReceptionKeyManagerBufferLength*2; i++ {
		newKm := *NewManager(baseKey, nil, nil,
			partner, false, 12, 10, 10)

		newKm.GenerateKeys(grp, userID)
		kmArray = append(kmArray, newKm)

		toDelete := aBuffer.push(&newKm)
		println("delete %v", toDelete)
		if i < ReceptionKeyManagerBufferLength {
			if len(toDelete) != 0 {
				//ERROR should have something
				t.Errorf("Error Nothing Should Be Returned to be deleted since" +
					" keybuffer should be filling up from empty state")
			}

			if &newKm != aBuffer.getCurrentReceptionKeyManager() {
				t.Errorf("Error incorrect Keymanager receieved from buffer.")
			}

		} else {
			if len(toDelete) == 0 {
				t.Errorf("Error not returning old keymanager to properly be disposed of")
			}

			if &newKm != aBuffer.getCurrentReceptionKeyManager() {
				t.Errorf("Error incorrect Keymanager receieved from buffer after its been filled up.")
			}
		}

	}

	if &kmArray[0] == &kmArray[1] {
		t.Errorf("Error tests fail because we are not creating a new Keymanager")
	}

}

//test that loc is always circular and outputted value is what is expected
func TestReceptionKeyManagerBuffer_getCurrentLoc(t *testing.T) {
	aBuffer := NewReceptionKeyManagerBuffer()

	if aBuffer.getCurrentLoc() != 0 {
		// Error location is not initialized as zero
		t.Errorf("Error ReceptionKeyManagerBuffer Loc not initialized to zero")
	}

	for i := 0; i < ReceptionKeyManagerBufferLength*2; i++ {

		aBuffer.push(&KeyManager{})

		if aBuffer.getCurrentLoc() != aBuffer.loc {
			//error mismatch between actual loc and returned loc
			t.Errorf("Error ReceptionKeyManagerBuffer Loc mismatch with Getfunction")
		}

		if aBuffer.loc > ReceptionKeyManagerBufferLength || aBuffer.loc < 0 {
			//Error Buffer Out of bounds
			t.Errorf("Error ReceptionKeyManagerBuffer Loc out of bounds error")
		}

		if aBuffer.loc != (i % ReceptionKeyManagerBufferLength) {
			//Error location is not circular

			t.Errorf("Error ReceptionKeyManagerBuffer Loc is not circular")
		}
	}

}

func TestReceptionKeyManagerBuffer_getCurrentReceptionKeyManager(t *testing.T) {
	aBuffer := NewReceptionKeyManagerBuffer()
	testManager := &KeyManager{}
	aBuffer.push(testManager)

	if aBuffer.getCurrentReceptionKeyManager() != testManager {
		t.Errorf("Error this is not the same manager pushed in.")
	}
}

func TestNewReceptionKeyManagerBuffer(t *testing.T) {
	aBuffer := NewReceptionKeyManagerBuffer()

	if aBuffer == nil {
		t.Errorf("Error creating new reception keymanager buffer returning nil")
	}
}

func TestReceptionKeyManagerBuffer_Gob(t *testing.T) {
	aBuffer := NewReceptionKeyManagerBuffer()
	grp := initGroup()
	baseKey := grp.NewInt(57)
	partner := id.NewIdFromUInt(14, id.User, t)
	userID := id.NewIdFromUInt(18, id.User, t)

	newKm := *NewManager(baseKey, nil,
		nil, partner,
		false, 12, 10, 10)

	newKm.GenerateKeys(grp, userID)

	aBuffer.push(&newKm)

	var byteBuf bytes.Buffer

	enc := gob.NewEncoder(&byteBuf)
	dec := gob.NewDecoder(&byteBuf)

	err := enc.Encode(aBuffer)

	if err != nil {
		t.Errorf("Failed to encode GOB KeyManagerBuffer: %s", err)
	}

	newBuffer := NewReceptionKeyManagerBuffer()
	err = dec.Decode(&newBuffer)
	if err != nil {
		t.Errorf("Failed to decode GOB KeyManagerBuffer: %s", err)
	}
}
