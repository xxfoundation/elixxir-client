///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package user

import (
	"gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

type User struct {
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

func (u User) GetContact() contact.Contact {
	return contact.Contact{
		ID:       u.ReceptionID.DeepCopy(),
		DhPubKey: u.E2eDhPublicKey,
		Facts:    make([]fact.Fact, 0),
	}
}

func NewUserFromProto(proto *Proto) User {
	return User{
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

func NewUserFromBackup(backup *backup.Backup) User {
	return User{
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
