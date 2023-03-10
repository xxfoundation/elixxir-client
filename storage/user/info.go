////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/rsa"
	oldrsa "gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type Proto struct {
	//General Identity
	TransmissionID   *id.ID
	TransmissionSalt []byte
	TransmissionRSA  *oldrsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     *oldrsa.PrivateKey
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
	TransmissionRSA  rsa.PrivateKey
	ReceptionID      *id.ID
	ReceptionSalt    []byte
	ReceptionRSA     rsa.PrivateKey
	Precanned        bool
	// Timestamp in which user has registered with the network
	RegistrationTimestamp int64

	//e2e Identity
	E2eDhPrivateKey *cyclic.Int
	E2eDhPublicKey  *cyclic.Int
}

func NewUserFromProto(proto *Proto) Info {
	sch := rsa.GetScheme()
	return Info{
		TransmissionID:        proto.TransmissionID,
		TransmissionSalt:      proto.TransmissionSalt,
		TransmissionRSA:       sch.Convert(&proto.TransmissionRSA.PrivateKey),
		ReceptionID:           proto.ReceptionID,
		ReceptionSalt:         proto.ReceptionSalt,
		ReceptionRSA:          sch.Convert(&proto.ReceptionRSA.PrivateKey),
		Precanned:             proto.Precanned,
		RegistrationTimestamp: proto.RegistrationTimestamp,
		E2eDhPrivateKey:       proto.E2eDhPrivateKey,
		E2eDhPublicKey:        proto.E2eDhPublicKey,
	}
}

func NewUserFromBackup(backup *backup.Backup) Info {
	sch := rsa.GetScheme()
	return Info{
		TransmissionID:        backup.TransmissionIdentity.ComputedID,
		TransmissionSalt:      backup.TransmissionIdentity.Salt,
		TransmissionRSA:       sch.Convert(&backup.TransmissionIdentity.RSASigningPrivateKey.PrivateKey),
		ReceptionID:           backup.ReceptionIdentity.ComputedID,
		ReceptionSalt:         backup.ReceptionIdentity.Salt,
		ReceptionRSA:          sch.Convert(&backup.ReceptionIdentity.RSASigningPrivateKey.PrivateKey),
		Precanned:             false,
		RegistrationTimestamp: backup.RegistrationTimestamp,
		E2eDhPrivateKey:       backup.ReceptionIdentity.DHPrivateKey,
		E2eDhPublicKey:        backup.ReceptionIdentity.DHPublicKey,
	}
}

func (u *User) PortableUserInfo() Info {
	ci := u.CryptographicIdentity
	return Info{
		TransmissionID:        ci.GetTransmissionID().DeepCopy(),
		TransmissionSalt:      copySlice(ci.GetTransmissionSalt()),
		TransmissionRSA:       ci.GetTransmissionRSA(),
		ReceptionID:           ci.GetReceptionID().DeepCopy(),
		RegistrationTimestamp: u.GetRegistrationTimestamp().UnixNano(),
		ReceptionSalt:         copySlice(ci.GetReceptionSalt()),
		ReceptionRSA:          ci.GetReceptionRSA(),
		Precanned:             ci.IsPrecanned(),
		E2eDhPrivateKey:       ci.GetE2eDhPrivateKey(),
		E2eDhPublicKey:        ci.GetE2eDhPublicKey(),
	}

}

func copySlice(s []byte) []byte {
	n := make([]byte, len(s))
	copy(n, s)
	return n
}
