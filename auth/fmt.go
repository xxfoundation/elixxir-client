///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	util "gitlab.com/elixxir/client/storage/utility"
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
	sidHpubkey []byte
	salt       []byte
	ecrPayload []byte
}

func newBaseFormat(payloadSize, pubkeySize, sidHPubkeySize int ) baseFormat {
	// NOTE: sidhPubKey needs an extra byte to hold the variant setting
	total := pubkeySize + sidHPubkeySize + 1 + saltSize
	if payloadSize < total {
		jww.FATAL.Panicf("Size of baseFormat is too small (%d), must be big " +
			"enough to contain public key (%d) sidHPublicKey (%d + 1) and salt (%d) " +
			"which totals to %d", payloadSize, pubkeySize, sidHPubkeySize, saltSize,
			total)
	}

	jww.INFO.Printf("Empty Space RequestAuth: %d", payloadSize-total)

	f := buildBaseFormat(make([]byte, payloadSize), pubkeySize,
		sidHPubkeySize)

	return f
}

func buildBaseFormat(data []byte, pubkeySize, sidHPubkeySize int) baseFormat {
	f := baseFormat{
		data: data,
	}

	start := 0
	end := pubkeySize
	f.pubkey = f.data[:end]

	start = end
	end = start + sidHPubkeySize + 1
	f.sidHpubkey = f.data[start:end]

	start = end
	end = start + saltSize
	f.salt = f.data[start:end]

	start = end
	f.ecrPayload = f.data[start:]
	return f
}

func unmarshalBaseFormat(b []byte, pubkeySize, sidHPubkeySize int) (baseFormat, error) {
	if len(b) < pubkeySize+saltSize {
		return baseFormat{}, errors.New("Received baseFormat too small")
	}

	return buildBaseFormat(b, pubkeySize, sidHPubkeySize), nil
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

func (f baseFormat) SetSidHPubKey(pubKey *sidh.PublicKey) {
	f.sidHpubkey[0] = byte(pubKey.Variant())
	pubKey.Export(f.sidHpubkey[1:])
}

func (f baseFormat) GetSidhPubKey() (*sidh.PublicKey, error) {
	variant := sidh.KeyVariant(f.sidHpubkey[0])
	pubKey := util.NewSIDHPublicKey(variant)
	err := pubKey.Import(f.sidHpubkey[1:])
	return pubKey, err
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
