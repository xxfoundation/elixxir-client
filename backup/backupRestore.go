////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// FIXME: This is placeholder, there's got to be a better place to put
// backup restoration than inside messenger.

package backup

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/client/xxdk"
	cryptoBackup "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// NewClientFromBackup constructs a new E2e from an encrypted
// backup. The backup is decrypted using the backupPassphrase. On
// success a successful client creation, the function will return a
// JSON encoded list of the E2E partners contained in the backup and a
// json-encoded string containing parameters stored in the backup
func NewClientFromBackup(ndfJSON, storageDir string, sessionPassword,
	backupPassphrase []byte, backupFileContents []byte) (
	xxdk.ReceptionIdentity, []*id.ID, string, error) {

	backUp := &cryptoBackup.Backup{}
	err := backUp.Decrypt(string(backupPassphrase), backupFileContents)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", errors.WithMessage(err,
			"Failed to unmarshal decrypted client contents.")
	}

	usr := user.NewUserFromBackup(backUp)

	def, err := xxdk.ParseNDF(ndfJSON)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	cmixGrp, e2eGrp := xxdk.DecodeGroups(def)

	// Note we do not need registration here
	storageSess, err := xxdk.CheckVersionAndSetupStorage(def, storageDir,
		sessionPassword, usr, cmixGrp, e2eGrp,
		backUp.RegistrationCode)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	identity, err := xxdk.BuildReceptionIdentity(usr.ReceptionID,
		usr.ReceptionSalt, usr.ReceptionRSA, e2eGrp, usr.E2eDhPrivateKey)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	storageSess.SetReceptionRegistrationValidationSignature(
		backUp.ReceptionIdentity.RegistrarSignature)
	storageSess.SetTransmissionRegistrationValidationSignature(
		backUp.TransmissionIdentity.RegistrarSignature)
	storageSess.SetRegistrationTimestamp(backUp.RegistrationTimestamp)

	//move the registration state to indicate registered with
	// registration on proto client
	err = storageSess.ForwardRegistrationStatus(
		storage.PermissioningComplete)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	privkey := usr.E2eDhPrivateKey

	//initialize the e2e storage
	err = e2e.Init(storageSess.GetKV(), usr.ReceptionID, privkey, e2eGrp,
		rekey.GetDefaultParams())
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	udInfo := backUp.UserDiscoveryRegistration
	var username, email, phone fact.Fact
	for _, f := range udInfo.FactList {
		switch f.T {
		case fact.Email:
			email = f
		case fact.Username:
			username = f
		case fact.Phone:
			phone = f
		}
	}
	err = ud.InitStoreFromBackup(storageSess.GetKV(), username, email, phone)
	return identity, backUp.Contacts.Identities, backUp.JSONParams, err
}
