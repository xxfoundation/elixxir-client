///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type Proto struct {
	//General Identity
	TransmissionID   *id.ID
	TransmissionSalt []byte
	TransmissionRSA  *rsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     *rsa.PrivateKey
	Precanned        bool
	// Timestamp in which user has registered with the network
	RegistrationTimestamp int64

	RegCode string

	TransmissionRegValidationSig []byte
	ReceptionRegValidationSig    []byte

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}

type Info struct {
	//General Identity
	TransmissionID   *id.ID
	TransmissionSalt []byte
	TransmissionRSA  *rsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     *rsa.PrivateKey
	Precanned        bool
	// Timestamp in which user has registered with the network
	RegistrationTimestamp int64

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}

func NewUserFromProto(proto *Proto) Info {
	return Info{
		TransmissionID:        proto.TransmissionID,
		TransmissionSalt:      proto.TransmissionSalt,
		TransmissionRSA:       proto.TransmissionRSA,
		ReceptionID:           proto.ReceptionID,
		ReceptionSalt:         proto.ReceptionSalt,
		ReceptionRSA:          proto.ReceptionRSA,
		Precanned:             proto.Precanned,
		RegistrationTimestamp: proto.RegistrationTimestamp,
		E2eDhPrivateKey:       proto.E2eDhPrivateKey,
		E2eDhPublicKey:        proto.E2eDhPublicKey,
	}
}

func NewUserFromBackup(backup *backup.Backup) Info {
	return Info{
		TransmissionID:        backup.TransmissionIdentity.ComputedID,
		TransmissionSalt:      backup.TransmissionIdentity.Salt,
		TransmissionRSA:       backup.TransmissionIdentity.RSASigningPrivateKey,
		ReceptionID:           backup.ReceptionIdentity.ComputedID,
		ReceptionSalt:         backup.ReceptionIdentity.Salt,
		ReceptionRSA:          backup.ReceptionIdentity.RSASigningPrivateKey,
		Precanned:             false,
		RegistrationTimestamp: backup.RegistrationTimestamp,
		E2eDhPrivateKey:       backup.ReceptionIdentity.DHPrivateKey,
		E2eDhPublicKey:        backup.ReceptionIdentity.DHPublicKey,
	}
}

func (u *User) PortableUserInfo() Info {
	ci := u.CryptographicIdentity
	pub, priv := u.loadLegacyDHKeys()
	return Info{
		TransmissionID:        ci.GetTransmissionID().DeepCopy(),
		TransmissionSalt:      copySlice(ci.GetTransmissionSalt()),
		TransmissionRSA:       ci.GetTransmissionRSA(),
		ReceptionID:           ci.GetReceptionID().DeepCopy(),
		RegistrationTimestamp: u.GetRegistrationTimestamp().UnixNano(),
		ReceptionSalt:         copySlice(ci.GetReceptionSalt()),
		ReceptionRSA:          ci.GetReceptionRSA(),
		Precanned:             ci.IsPrecanned(),
		E2eDhPrivateKey:       priv,
		E2eDhPublicKey:        pub,
	}

}

// loadLegacyDHKeys attempts to load DH Keys from legacy storage. It
// prints a warning to the log as users should be using ReceptionIdentity
// instead of PortableUserInfo
func (u *User) loadLegacyDHKeys() (pub, priv *cyclic.Int) {
	jww.WARN.Printf("loading legacy e2e keys in cmix layer, please " +
		"update your code to use xxdk.ReceptionIdentity")
	// Legacy package prefixes and keys, see e2e/ratchet/storage.go
	packagePrefix := "e2eSession"
	pubKeyKey := "DhPubKey"
	privKeyKey := "DhPrivKey"

	kv := u.kv.Prefix(packagePrefix)

	privKey, err := utility.LoadCyclicKey(kv, privKeyKey)
	if err != nil {
		jww.ERROR.Printf("Failed to load e2e DH private key: %v", err)
		return nil, nil
	}

	pubKey, err := utility.LoadCyclicKey(kv, pubKeyKey)
	if err != nil {
		jww.ERROR.Printf("Failed to load e2e DH public key: %v", err)
		return nil, nil
	}

	return pubKey, privKey
}

func copySlice(s []byte) []byte {
	n := make([]byte, len(s))
	copy(n, s)
	return n
}
