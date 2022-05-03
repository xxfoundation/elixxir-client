///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentCryptographicIdentityVersion = 0
const cryptographicIdentityKey = "cryptographicIdentity"

type CryptographicIdentity struct {
	transmissionID     *id.ID
	transmissionSalt   []byte
	transmissionRsaKey *rsa.PrivateKey
	receptionID        *id.ID
	receptionSalt      []byte
	receptionRsaKey    *rsa.PrivateKey
	isPrecanned        bool
}

type ciDisk struct {
	TransmissionID     *id.ID
	TransmissionSalt   []byte
	TransmissionRsaKey *rsa.PrivateKey
	ReceptionID        *id.ID
	ReceptionSalt      []byte
	ReceptionRsaKey    *rsa.PrivateKey
	IsPrecanned        bool
}

func newCryptographicIdentity(transmissionID, receptionID *id.ID,
	transmissionSalt, receptionSalt []byte,
	transmissionRsa, receptionRsa *rsa.PrivateKey,
	isPrecanned bool, kv *versioned.KV) *CryptographicIdentity {

	ci := &CryptographicIdentity{
		transmissionID:     transmissionID,
		transmissionSalt:   transmissionSalt,
		transmissionRsaKey: transmissionRsa,
		receptionID:        receptionID,
		receptionSalt:      receptionSalt,
		receptionRsaKey:    receptionRsa,
		isPrecanned:        isPrecanned,
	}

	if err := ci.save(kv); err != nil {
		jww.FATAL.Panicf("Failed to store the new Cryptographic"+
			" Identity: %s", err)
	}

	return ci
}

func loadCryptographicIdentity(kv *versioned.KV) (*CryptographicIdentity, error) {
	obj, err := kv.Get(cryptographicIdentityKey,
		currentCryptographicIdentityVersion)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get user "+
			"cryptographic identity from EKV")
	}

	var resultBuffer bytes.Buffer
	result := &CryptographicIdentity{}
	decodable := &ciDisk{}

	resultBuffer.Write(obj.Data)
	dec := gob.NewDecoder(&resultBuffer)
	err = dec.Decode(decodable)

	if decodable != nil {
		result.isPrecanned = decodable.IsPrecanned
		result.receptionRsaKey = decodable.ReceptionRsaKey
		result.transmissionRsaKey = decodable.TransmissionRsaKey
		result.transmissionSalt = decodable.TransmissionSalt
		result.transmissionID = decodable.TransmissionID
		result.receptionID = decodable.ReceptionID
		result.receptionSalt = decodable.ReceptionSalt
	}

	return result, err
}

func (ci *CryptographicIdentity) save(kv *versioned.KV) error {
	var userDataBuffer bytes.Buffer

	encodable := &ciDisk{
		TransmissionID:     ci.transmissionID,
		TransmissionSalt:   ci.transmissionSalt,
		TransmissionRsaKey: ci.transmissionRsaKey,
		ReceptionID:        ci.receptionID,
		ReceptionSalt:      ci.receptionSalt,
		ReceptionRsaKey:    ci.receptionRsaKey,
		IsPrecanned:        ci.isPrecanned,
	}

	enc := gob.NewEncoder(&userDataBuffer)
	err := enc.Encode(encodable)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   currentCryptographicIdentityVersion,
		Timestamp: netTime.Now(),
		Data:      userDataBuffer.Bytes(),
	}

	return kv.Set(cryptographicIdentityKey,
		currentCryptographicIdentityVersion, obj)
}

func (ci *CryptographicIdentity) GetTransmissionID() *id.ID {
	return ci.transmissionID.DeepCopy()
}

func (ci *CryptographicIdentity) GetTransmissionSalt() []byte {
	return ci.transmissionSalt
}

func (ci *CryptographicIdentity) GetReceptionID() *id.ID {
	return ci.receptionID.DeepCopy()
}

func (ci *CryptographicIdentity) GetReceptionSalt() []byte {
	return ci.receptionSalt
}

func (ci *CryptographicIdentity) GetReceptionRSA() *rsa.PrivateKey {
	return ci.receptionRsaKey
}

func (ci *CryptographicIdentity) GetTransmissionRSA() *rsa.PrivateKey {
	return ci.transmissionRsaKey
}

func (ci *CryptographicIdentity) IsPrecanned() bool {
	return ci.isPrecanned
}
