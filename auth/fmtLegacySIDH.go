////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	sidhinterface "gitlab.com/elixxir/client/interfaces/sidh"
	util "gitlab.com/elixxir/client/storage/utility"
)

const requestLegacySIDHFmtVersion = 1

//Basic Format//////////////////////////////////////////////////////////////////
func newLegacySIDHBaseFormat(payloadSize, pubkeySize int) baseFormat {
	total := pubkeySize
	// Size of sidh pubkey
	total += sidhinterface.PubKeyByteSize + 1
	// Size of version
	total += 1
	if payloadSize < total {
		jww.FATAL.Panicf("Size of baseFormat is too small "+
			"(%d), must be big enough to contain public "+
			"key (%d) and sidh key (%d) and version "+
			"which totals to %d", payloadSize, pubkeySize,
			sidhinterface.PubKeyByteSize+1, total)
	}

	jww.INFO.Printf("Empty Space RequestAuth: %d", payloadSize-total)

	f := buildBaseFormat(make([]byte, payloadSize), pubkeySize)
	f.version[0] = requestLegacySIDHFmtVersion

	return f
}

//Encrypted Format//////////////////////////////////////////////////////////////

type ecrLegacySIDHFormat struct {
	data       []byte
	ownership  []byte
	sidHpubkey []byte
	payload    []byte
}

func newLegacySIDHEcrFormat(size int) ecrLegacySIDHFormat {
	if size < (ownershipSize + sidhinterface.PubKeyByteSize + 1) {
		jww.FATAL.Panicf("Size too small to hold")
	}

	f := buildLegacySIDHEcrFormat(make([]byte, size))

	return f

}

func buildLegacySIDHEcrFormat(data []byte) ecrLegacySIDHFormat {
	f := ecrLegacySIDHFormat{
		data: data,
	}

	start := 0
	end := ownershipSize
	f.ownership = f.data[start:end]

	start = end
	end = start + sidhinterface.PubKeyByteSize + 1
	f.sidHpubkey = f.data[start:end]

	start = end
	f.payload = f.data[start:]
	return f
}

func unmarshalLegacySIDHEcrFormat(b []byte) (ecrLegacySIDHFormat, error) {
	if len(b) < ownershipSize {
		return ecrLegacySIDHFormat{},
			errors.New("Received ecr baseFormat too small")
	}

	return buildLegacySIDHEcrFormat(b), nil
}

func (f ecrLegacySIDHFormat) Marshal() []byte {
	return f.data
}

func (f ecrLegacySIDHFormat) GetOwnership() []byte {
	return f.ownership
}

func (f ecrLegacySIDHFormat) SetOwnership(ownership []byte) {
	if len(ownership) != ownershipSize {
		jww.FATAL.Panicf("ownership proof is the wrong size")
	}

	copy(f.ownership, ownership)
}

func (f ecrLegacySIDHFormat) SetSidHPubKey(pubKey *sidh.PublicKey) {
	f.sidHpubkey[0] = byte(pubKey.Variant())
	pubKey.Export(f.sidHpubkey[1:])
}

func (f ecrLegacySIDHFormat) GetSidhPubKey() (*sidh.PublicKey, error) {
	variant := sidh.KeyVariant(f.sidHpubkey[0])
	pubKey := util.NewSIDHPublicKey(variant)
	err := pubKey.Import(f.sidHpubkey[1:])
	return pubKey, err
}

func (f ecrLegacySIDHFormat) GetPayload() []byte {
	return f.payload
}

func (f ecrLegacySIDHFormat) PayloadLen() int {
	return len(f.payload)
}

func (f ecrLegacySIDHFormat) SetPayload(p []byte) {
	if len(p) != len(f.payload) {
		jww.FATAL.Panicf("Payload is the wrong length")
	}

	copy(f.payload, p)
}
