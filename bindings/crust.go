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
)

// UploadBackup will upload the file provided to the distributed file server.
// This will return a UploadSuccessReport, which provides data on the status of the
// upload. The file may be recovered using RecoverBackup.
func (c *Client) UploadBackup(file []byte, udManager *UserDiscovery) ([]byte, error) {
	privateKey := c.api.GetUser().ReceptionRSA
	uploadSuccessReport, err := crust.UploadBackup(file, privateKey, udManager.ud)
	if err != nil {
		return nil, err
	}

	return json.Marshal(uploadSuccessReport)
}

// RecoverBackup retrieves the backup file uploaded to the distributed file
// server. The user must have called UploadBackup successfully for a proper
// file recover.
func (c *Client) RecoverBackup(username string) ([]byte, error) {
	usernameHash := crust2.HashUsername(username)

	return crust.RecoverBackup(usernameHash)
}
