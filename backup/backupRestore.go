////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/rekey"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/ud"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoBackup "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// NewCmixFromBackup initializes a new e2e storage from an encrypted
// backup. The backup is decrypted using the backupPassphrase. On
// a successful client creation, the function will return a
// JSON encoded list of the E2E partners contained in the backup and a
// json-encoded string containing parameters stored in the backup
func NewCmixFromBackup(ndfJSON, storageDir, backupPassphrase string,
	sessionPassword []byte, backupFileContents []byte) ([]*id.ID,
	string, error) {

	rngStreamGen := fastRNG.NewStreamGenerator(12, 1024,
		csprng.NewSystemRNG)
	rngStream := rngStreamGen.GetStream()
	defer rngStream.Close()

	backUp := &cryptoBackup.Backup{}
	err := backUp.Decrypt(backupPassphrase, backupFileContents)
	if err != nil {
		return nil, "", errors.WithMessage(err,
			"Failed to unmarshal decrypted client contents.")
	}

	jww.INFO.Printf("Decrypted backup ID to Restore: %v",
		backUp.ReceptionIdentity.ComputedID)

	userInfo := user.NewUserFromBackup(backUp)

	def, err := xxdk.ParseNDF(ndfJSON)
	if err != nil {
		return nil, "", err
	}

	cmixGrp, e2eGrp := xxdk.DecodeGroups(def)

	kv, err := xxdk.LocalKV(storageDir, sessionPassword, rngStreamGen)
	if err != nil {
		return nil, "", err
	}

	// Note we do not need registration here
	storageSess, err := xxdk.CheckVersionAndSetupStorage(def, kv, userInfo,
		cmixGrp, e2eGrp, backUp.RegistrationCode, rngStreamGen)
	if err != nil {
		return nil, "", err
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
		return nil, "", err
	}

	privKey := userInfo.E2eDhPrivateKey

	//initialize the e2e storage
	err = e2e.Init(storageSess.GetKV(), userInfo.ReceptionID, privKey, e2eGrp,
		rekey.GetDefaultParams())
	if err != nil {
		return nil, "", err
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
	return backUp.Contacts.Identities, backUp.JSONParams, err
}
