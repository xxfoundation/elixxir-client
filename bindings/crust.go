////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/partnerships/crust"
	crust2 "gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/utils"
)

// UploadBackup will upload the file provided to the distributed file server.
// This will return a UploadSuccessReport, which provides data on the status of
// the upload. The file may be recovered using RecoverBackup.
//
// Parameters:
//  - filePath - the path to the backup file that will be uploaded to the
//    backup server.
//  - udManager - the UserDiscovery object.
//  - receptionRsaPrivateKey - the PEM encoded reception RSA private key. This
//    can be retrieved via Client.GetUser.GetReceptionRSAPrivateKeyPem.
func UploadBackup(filePath string, udManager *UserDiscovery,
	receptionRsaPrivateKey []byte) ([]byte, error) {

	privateKey, err := rsa.LoadPrivateKeyFromPem(receptionRsaPrivateKey)
	if err != nil {
		return nil, err
	}

	file, err := utils.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	backupFile := crust.NewBackupFile(filePath, file)
	uploadSuccessReport, err := crust.UploadBackup(backupFile, privateKey,
		udManager.ud)
	if err != nil {
		return nil, err
	}

	return json.Marshal(uploadSuccessReport)
}

// RecoverBackup retrieves the backup file uploaded to the distributed file
// server. The user must have called UploadBackup successfully for a proper
// file recover.
func RecoverBackup(username string) ([]byte, error) {
	usernameHash := crust2.HashUsername(username)

	return crust.RecoverBackup(string(usernameHash))
}
