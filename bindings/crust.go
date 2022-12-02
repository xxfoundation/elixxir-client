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
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/utils"
)

// UploadSuccessReport is the report received when calling UploadBackup.
// Example JSON:
//   {
//     "requestid":"f98fa83e-209c-42b1-be05-011e11643162-1670008805297",
//     "status":"queued",
//     "created":"2022-12-02T19:20:05Z",
//     "pin":{
//        "cid":"QmUe8UmQ5iEKHQNp9z5HN7fXmcf666zUuPb6N24oiGq31G",
//        "name":"LoremIpsum.txt",
//        "origins":[
//        ]
//     },
//     "delegates":[
//        "/ip4/183.131.193.198/tcp/14001/p2p/12D3KooWMcAHcs97R49PLZjGUKDbP1fr9iijeepod8fkktHTLCgN"
//     ],
//     "info":{}
//  }
type UploadSuccessReport struct {
	crust.UploadSuccessReport
}

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

	report := UploadSuccessReport{*uploadSuccessReport}

	return json.Marshal(report)
}

// RecoverBackup retrieves the backup file uploaded to the distributed file
// server. The user must have called UploadBackup successfully for a proper
// file recover.
func RecoverBackup(username string) ([]byte, error) {
	return crust.RecoverBackup(username)
}
