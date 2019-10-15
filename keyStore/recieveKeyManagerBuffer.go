package keyStore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/pkg/errors"
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

func (rkmb *ReceptionKeyManagerBuffer) GobEncode () ([]byte, error){

	//get rid of nils for encoding
	var bufferSlice []*KeyManager

	for i:=0; i < len(rkmb.managers); i++{
		j := ((rkmb.loc + i) % len(rkmb.managers))
		if rkmb.managers[j] != nil{
			bufferSlice = append(bufferSlice, rkmb.managers[j])
		}


	}

	anon := struct {
		Managers []*KeyManager
		Loc int
	}{
		bufferSlice,
		rkmb.loc,
	}


	var encodeBytes bytes.Buffer

	enc := gob.NewEncoder(&encodeBytes)

	err := enc.Encode(anon)

	if err != nil {
		err = errors.New(fmt.Sprintf("Could not encode Reception Keymanager Buffer: %s",
			err.Error()))
		return nil, err
	}
	return encodeBytes.Bytes(), nil

}

func (rkmb *ReceptionKeyManagerBuffer) GobDecode (in []byte) error{

	anon := struct {
		Managers []*KeyManager
		Loc int
	}{

	}

	var buf bytes.Buffer

	// Write bytes to the buffer
	buf.Write(in)

	dec := gob.NewDecoder(&buf)

	err := dec.Decode(&anon)

	if err != nil{
		err = errors.New(fmt.Sprintf("Could not Decode Reception Keymanager Buffer: %s", err.Error()))
		return err
	}


	rkmb.loc = anon.Loc

	for i:=0; i < len(anon.Managers); i++{
		j := ((anon.Loc + i) % len(rkmb.managers))
		rkmb.managers[j] = anon.Managers[i]
	}

	return nil
}