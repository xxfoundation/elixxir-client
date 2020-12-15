///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

//Basic Format//////////////////////////////////////////////////////////////////
const saltSize = 32

type baseFormat struct {
	data       []byte
	pubkey     []byte
	salt       []byte
	ecrPayload []byte
}

func newBaseFormat(payloadSize, pubkeySize int) baseFormat {

	if payloadSize < pubkeySize+saltSize {
		jww.FATAL.Panicf("Size of baseFormat is too small, must be big " +
			"enough to contain public key and salt")
	}

	f := buildBaseFormat(make([]byte, payloadSize), pubkeySize)

	return f
}

func buildBaseFormat(data []byte, pubkeySize int) baseFormat {
	f := baseFormat{
		data: data,
	}

	f.pubkey = f.data[:pubkeySize]
	f.salt = f.data[pubkeySize : pubkeySize+saltSize]
	f.ecrPayload = f.data[pubkeySize+saltSize:]
	return f
}

func unmarshalBaseFormat(b []byte, pubkeySize int) (baseFormat, error) {
	if len(b) < pubkeySize+saltSize {
		return baseFormat{}, errors.New("Received baseFormat too small")
	}

	return buildBaseFormat(b, pubkeySize), nil
}

func (f baseFormat) Marshal() []byte {
	return f.data
}

func (f baseFormat) GetPubKey(grp *cyclic.Group) *cyclic.Int {
	return grp.NewIntFromBytes(f.pubkey)
}

func (f baseFormat) SetPubKey(pubKey *cyclic.Int) {
	pubKeyBytes := pubKey.LeftpadBytes(uint64(len(f.pubkey)))
	copy(f.pubkey, pubKeyBytes)
}

func (f baseFormat) GetSalt() []byte {
	return f.salt
}

func (f baseFormat) SetSalt(salt []byte) {
	if len(salt) != saltSize {
		jww.FATAL.Panicf("Salt incorrect size")
	}

	copy(f.salt, salt)
}

func (f baseFormat) GetEcrPayload() []byte {
	return f.ecrPayload
}

func (f baseFormat) GetEcrPayloadLen() int {
	return len(f.ecrPayload)
}

func (f baseFormat) SetEcrPayload(ecr []byte) {
	if len(ecr) != len(f.ecrPayload) {
		jww.FATAL.Panicf("Passed ecr payload incorrect lengh. Expected:"+
			" %v, Recieved: %v", len(f.ecrPayload), len(ecr))
	}

	copy(f.ecrPayload, ecr)
}

//Encrypted Format//////////////////////////////////////////////////////////////
const ownershipSize = 32

type ecrFormat struct {
	data      []byte
	ownership []byte
	payload   []byte
}

func newEcrFormat(size int) ecrFormat {
	if size < ownershipSize {
		jww.FATAL.Panicf("Size too small to hold")
	}

	f := buildEcrFormat(make([]byte, size))

	return f

}

func buildEcrFormat(data []byte) ecrFormat {
	f := ecrFormat{
		data: data,
	}

	f.ownership = f.data[:ownershipSize]
	f.payload = f.data[ownershipSize:]
	return f
}

func unmarshalEcrFormat(b []byte) (ecrFormat, error) {
	if len(b) < ownershipSize {
		return ecrFormat{}, errors.New("Received ecr baseFormat too small")
	}

	return buildEcrFormat(b), nil
}

func (f ecrFormat) Marshal() []byte {
	return f.data
}

func (f ecrFormat) GetOwnership() []byte {
	return f.ownership
}

func (f ecrFormat) SetOwnership(ownership []byte) {
	if len(ownership) != ownershipSize {
		jww.FATAL.Panicf("ownership proof is the wrong size")
	}

	copy(f.ownership, ownership)
}

func (f ecrFormat) GetPayload() []byte {
	return f.payload
}

func (f ecrFormat) PayloadLen() int {
	return len(f.payload)
}

func (f ecrFormat) SetPayload(p []byte) {
	if len(p) != len(f.payload) {
		jww.FATAL.Panicf("Payload is the wrong length")
	}

	copy(f.payload, p)
}

//Request Format////////////////////////////////////////////////////////////////
type requestFormat struct {
	ecrFormat
	id         []byte
	msgPayload []byte
}

func newRequestFormat(ecrFmt ecrFormat) (requestFormat, error) {
	if len(ecrFmt.payload) < id.ArrIDLen {
		return requestFormat{}, errors.New("Payload is not long enough")
	}

	rf := requestFormat{
		ecrFormat: ecrFmt,
	}

	rf.id = rf.payload[:id.ArrIDLen]
	rf.msgPayload = rf.payload[id.ArrIDLen:]

	return rf, nil
}

func (rf requestFormat) GetID() (*id.ID, error) {
	return id.Unmarshal(rf.id)
}

func (rf requestFormat) SetID(myId *id.ID) {
	copy(rf.id, myId.Marshal())
}

func (rf requestFormat) SetMsgPayload(b []byte) {
	if len(b) > len(rf.msgPayload) {
		jww.FATAL.Panicf("Message Payload is too long")
	}

	copy(rf.msgPayload, b)
}

func (rf requestFormat) MsgPayloadLen() int {
	return len(rf.msgPayload)
}

func (rf requestFormat) GetMsgPayload() []byte {
	return rf.msgPayload
}
