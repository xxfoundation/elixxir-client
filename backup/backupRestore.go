////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
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

// NewClientFromBackup initializes a new e2e storage from an encrypted
// backup. The backup is decrypted using the backupPassphrase. On
// a successful client creation, the function will return a
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

	jww.INFO.Printf("Decrypted backup ID to Restore: %v",
		backUp.ReceptionIdentity.ComputedID)

	userInfo := user.NewUserFromBackup(backUp)

	def, err := xxdk.ParseNDF(ndfJSON)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	cmixGrp, e2eGrp := xxdk.DecodeGroups(def)

	// Note we do not need registration here
	storageSess, err := xxdk.CheckVersionAndSetupStorage(def, storageDir,
		sessionPassword, userInfo, cmixGrp, e2eGrp,
		backUp.RegistrationCode)
	if err != nil {
		return xxdk.ReceptionIdentity{}, nil, "", err
	}

	identity, err := xxdk.BuildReceptionIdentity(userInfo.ReceptionID,
		userInfo.ReceptionSalt, userInfo.ReceptionRSA, e2eGrp,
		userInfo.E2eDhPrivateKey)
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

	privKey := userInfo.E2eDhPrivateKey

	//initialize the e2e storage
	err = e2e.Init(storageSess.GetKV(), userInfo.ReceptionID, privKey, e2eGrp,
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
