package keyStore

import (
	"gitlab.com/elixxir/primitives/format"
)

const ReceptionKeyManagerBufferLength = 5

//This creates a circular buffer and initializes all the keymanagers to be nil at location zero.
func NewReceptionKeyManagerBuffer () *ReceptionKeyManagerBuffer{
	newBuffer :=  ReceptionKeyManagerBuffer{}
	newBuffer.loc = 0
	return &newBuffer
}

type ReceptionKeyManagerBuffer struct{
	managers [ReceptionKeyManagerBufferLength]*KeyManager
	loc int
}

// Push takes in a new keymanager obj, and adds it into our circular buffer of keymanagers,
// the keymanager obj passed in overwrites the keymanager in the buffer, and we have to return the existing
// keymanager if there is one back ot the parent so that the deletion can be handled.
func (rkmb *ReceptionKeyManagerBuffer) push (km *KeyManager) ([]format.Fingerprint) {
	deadkm := &KeyManager{}
	deadkm = nil
	if rkmb.managers[0] != nil {
		//Don't increment location if location 0 is empty first time around
		rkmb.loc = (rkmb.loc + 1) % ReceptionKeyManagerBufferLength
		deadkm = rkmb.managers[rkmb.loc]
	} else{

	}


	rkmb.managers[rkmb.loc] = km

	if deadkm == nil{
		return []format.Fingerprint{}
	}else{

		return append(deadkm.recvKeysFingerprint, deadkm.recvReKeysFingerprint...)

	}
}

func (rkmb *ReceptionKeyManagerBuffer) getCurrentReceptionKeyManager () *KeyManager {
	return rkmb.managers[rkmb.loc]
}

func (rkmb *ReceptionKeyManagerBuffer) getCurrentLoc() int {
	return rkmb.loc
}

func (rkmb *ReceptionKeyManagerBuffer) getReceptionKeyManagerAtLoc(n int) *KeyManager{
	return rkmb.managers[n % ReceptionKeyManagerBufferLength]
}

