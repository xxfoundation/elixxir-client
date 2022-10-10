////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const originalCryptographicIdentityVersion = 0
const currentCryptographicIdentityVersion = 1
const cryptographicIdentityKey = "cryptographicIdentity"

type CryptographicIdentity struct {
	transmissionID     *id.ID
	transmissionSalt   []byte
	transmissionRsaKey *rsa.PrivateKey
	receptionID        *id.ID
	receptionSalt      []byte
	receptionRsaKey    *rsa.PrivateKey
	isPrecanned        bool
	e2eDhPrivateKey    *cyclic.Int
	e2eDhPublicKey     *cyclic.Int
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

type ciDiskV1 struct {
	TransmissionID     *id.ID
	TransmissionSalt   []byte
	TransmissionRsaKey *rsa.PrivateKey
	ReceptionID        *id.ID
	ReceptionSalt      []byte
	ReceptionRsaKey    *rsa.PrivateKey
	IsPrecanned        bool
	E2eDhPrivateKey    []byte
	E2eDhPublicKey     []byte
}

func newCryptographicIdentity(transmissionID, receptionID *id.ID,
	transmissionSalt, receptionSalt []byte,
	transmissionRsa, receptionRsa *rsa.PrivateKey,
	isPrecanned bool, e2eDhPrivateKey, e2eDhPublicKey *cyclic.Int,
	kv *versioned.KV) *CryptographicIdentity {

	ci := &CryptographicIdentity{
		transmissionID:     transmissionID,
		transmissionSalt:   transmissionSalt,
		transmissionRsaKey: transmissionRsa,
		receptionID:        receptionID,
		receptionSalt:      receptionSalt,
		receptionRsaKey:    receptionRsa,
		isPrecanned:        isPrecanned,
		e2eDhPrivateKey:    e2eDhPrivateKey,
		e2eDhPublicKey:     e2eDhPublicKey,
	}

	if err := ci.save(kv); err != nil {
		jww.FATAL.Panicf("Failed to store the new Cryptographic"+
			" Identity: %s", err)
	}

	return ci
}

// loadOriginalCryptographicIdentity attempts to load the originalCryptographicIdentityVersion CryptographicIdentity
func loadOriginalCryptographicIdentity(kv *versioned.KV) (*CryptographicIdentity, error) {
	result := &CryptographicIdentity{}
	obj, err := kv.Get(cryptographicIdentityKey, originalCryptographicIdentityVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to get version %d user "+
			"cryptographic identity from EKV", originalCryptographicIdentityVersion)
	}
	var resultBuffer bytes.Buffer
	decodable := &ciDisk{}

	resultBuffer.Write(obj.Data)
	dec := gob.NewDecoder(&resultBuffer)
	err = dec.Decode(decodable)
	if err != nil {
		return nil, err
	}

	result.isPrecanned = decodable.IsPrecanned
	result.receptionRsaKey = decodable.ReceptionRsaKey
	result.transmissionRsaKey = decodable.TransmissionRsaKey
	result.transmissionSalt = decodable.TransmissionSalt
	result.transmissionID = decodable.TransmissionID
	result.receptionID = decodable.ReceptionID
	result.receptionSalt = decodable.ReceptionSalt
	return result, nil
}

func loadCryptographicIdentity(kv *versioned.KV) (*CryptographicIdentity, error) {
	result := &CryptographicIdentity{}
	obj, err := kv.Get(cryptographicIdentityKey,
		currentCryptographicIdentityVersion)
	if err != nil {
		result, err = loadOriginalCryptographicIdentity(kv)
		if err != nil {
			return nil, err
		}
		jww.WARN.Printf("Attempting to migrate cryptographic identity to new version...")
		// Populate E2E keys from legacy storage
		result.e2eDhPublicKey, result.e2eDhPrivateKey = loadLegacyDHKeys(kv)
		// Migrate to the new version in storage
		return result, result.save(kv)
	}

	decodable := &ciDiskV1{}
	err = json.Unmarshal(obj.Data, decodable)
	if err != nil {
		return nil, err
	}

	result.isPrecanned = decodable.IsPrecanned
	result.receptionRsaKey = decodable.ReceptionRsaKey
	result.transmissionRsaKey = decodable.TransmissionRsaKey
	result.transmissionSalt = decodable.TransmissionSalt
	result.transmissionID = decodable.TransmissionID
	result.receptionID = decodable.ReceptionID
	result.receptionSalt = decodable.ReceptionSalt

	result.e2eDhPrivateKey = &cyclic.Int{}
	err = result.e2eDhPrivateKey.UnmarshalJSON(decodable.E2eDhPrivateKey)
	if err != nil {
		return nil, err
	}
	result.e2eDhPublicKey = &cyclic.Int{}
	err = result.e2eDhPublicKey.UnmarshalJSON(decodable.E2eDhPublicKey)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// loadLegacyDHKeys attempts to load DH Keys from legacy storage. It
// prints a warning to the log as users should be using ReceptionIdentity
// instead of PortableUserInfo
func loadLegacyDHKeys(kv *versioned.KV) (pub, priv *cyclic.Int) {
	// Legacy package prefixes and keys, see e2e/ratchet/storage.go
	packagePrefix := "e2eSession"
	pubKeyKey := "DhPubKey"
	privKeyKey := "DhPrivKey"

	kvPrefix := kv.Prefix(packagePrefix)

	privKey, err := utility.LoadCyclicKey(kvPrefix, privKeyKey)
	if err != nil {
		jww.ERROR.Printf("Failed to load e2e DH private key: %v", err)
		return nil, nil
	}

	pubKey, err := utility.LoadCyclicKey(kvPrefix, pubKeyKey)
	if err != nil {
		jww.ERROR.Printf("Failed to load e2e DH public key: %v", err)
		return nil, nil
	}

	return pubKey, privKey
}

func (ci *CryptographicIdentity) save(kv *versioned.KV) error {
	dhPriv, err := ci.e2eDhPrivateKey.MarshalJSON()
	if err != nil {
		return err
	}
	dhPub, err := ci.e2eDhPublicKey.MarshalJSON()
	if err != nil {
		return err
	}

	encodable := &ciDiskV1{
		TransmissionID:     ci.transmissionID,
		TransmissionSalt:   ci.transmissionSalt,
		TransmissionRsaKey: ci.transmissionRsaKey,
		ReceptionID:        ci.receptionID,
		ReceptionSalt:      ci.receptionSalt,
		ReceptionRsaKey:    ci.receptionRsaKey,
		IsPrecanned:        ci.isPrecanned,
		E2eDhPrivateKey:    dhPriv,
		E2eDhPublicKey:     dhPub,
	}

	enc, err := json.Marshal(&encodable)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   currentCryptographicIdentityVersion,
		Timestamp: netTime.Now(),
		Data:      enc,
	}

	return kv.Set(cryptographicIdentityKey, obj)
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

func (ci *CryptographicIdentity) GetE2eDhPublicKey() *cyclic.Int {
	return ci.e2eDhPublicKey.DeepCopy()
}

func (ci *CryptographicIdentity) GetE2eDhPrivateKey() *cyclic.Int {
	return ci.e2eDhPrivateKey.DeepCopy()
}
