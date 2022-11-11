////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"net/http"
)

// Recovery URLs and tokens.
const (
	cidRequestURL    = "https://pin.crustcode.com/psa/value?key="
	recoveryUrl      = "https://crustipfs.xyz/ipfs/"
	recoveryAuthUser = "sub-5FqwjYoLQq9gcWHc6k5Ywtndx7CMiCQeKSvLzXojegvXcM9m"
	recoveryAuthPass = "0x20f6187988b621f97d404fefc3e491e99541f937c43bad25a74d694c2065f381d979bf4a43bb70fec653200b62aafdb9ac710875c9d28ea2c506d72d75cdc706"
)

// RecoveryResponse is the response from Restore.
type RecoveryResponse struct {
	// Value is the base64 encoded file that was backup up to the network.
	Value string
}

// cidResponse is the response from the requestCid call to the pinner.
type cidResponse struct {
	// Value is the CID associated with our username. This
	// value allows us to request the file from the network.
	Value string
}

// RecoverBackup retrieves the backup file uploaded to the distributed file
// server. The user must have called UploadBackup successfully for a proper
// file recover.
func RecoverBackup(usernameHash string) ([]byte, error) {
	cidResp, err := requestCid(usernameHash)
	if err != nil {
		return nil, errors.Errorf("failed to retrieve CID: %+v", err)
	}

	backupFile, err := requestBackupFile(cidResp)
	if err != nil {
		return nil, errors.Errorf("failed to retrieve backup file: %+v", err)
	}

	return backupFile, nil
}

// requestCid requests the CID associated with this username.
// This allows the user to request their backup file using requestBackupFile.
func requestCid(usernameHash string) (*cidResponse, error) {

	// Construct request to get CID
	req, err := http.NewRequest(http.MethodGet, cidRequestURL+usernameHash, nil)
	if err != nil {
		return nil, err
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	// Add auth header
	req.SetBasicAuth(recoveryAuthUser, recoveryAuthPass)

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Parse request
	cidResp := &cidResponse{}
	err = json.Unmarshal(responseData, cidResp)
	if err != nil {
		return nil, err
	}

	return cidResp, nil
}

// requestBackupFile sends the CID to the network to retrieve the backed up
// file.
func requestBackupFile(cid *cidResponse) ([]byte, error) {

	// Construct restore GET request
	req, err := http.NewRequest(http.MethodGet, recoveryUrl+cid.Value, nil)
	if err != nil {
		return nil, err
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	// Add auth header
	req.SetBasicAuth(recoveryAuthUser, recoveryAuthPass)

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Parse response
	recoveryResponse := &RecoveryResponse{}
	err = json.Unmarshal(responseData, recoveryResponse)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(recoveryResponse.Value)
}
