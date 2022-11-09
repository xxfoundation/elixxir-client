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
	cidRequestURL     = "https://pin.crustcode.com/psa/value?key="
	recoveryUrl       = "https://crustipfs.xyz/ipfs/"
	recoveryAuthToken = `c3ViLTVGcXdqWW9MUXE5Z2NXSGM2azVZd3RuZHg3Q01pQ1FlS1N2THpYb2plZ3ZYY005bToweDIwZjYxODc5ODhiNjIxZjk3ZDQwNGZlZmMzZTQ5MWU5OTU0MWY5MzdjNDNiYWQyNWE3NGQ2OTRjMjA2NWYzODFkOTc5YmY0YTQzYmI3MGZlYzY1MzIwMGI2MmFhZmRiOWFjNzEwODc1YzlkMjhlYTJjNTA2ZDcyZDc1Y2RjNzA2`
)

// cidResponse is the response from the requestCid call to the pinner.
type cidResponse struct {
	// Value is the CID associated with our username. This
	// value allows us to request the file from the network.
	Value string
}

// RecoveryResponse is the response from Restore.
type RecoveryResponse struct {
	// Value is the base64 encoded file that was backup up to the network.
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

	// Add header
	req.Header.Add(basicAuthHeader, recoveryAuthToken)

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Parse request
	cidResponse := &cidResponse{}
	err = json.Unmarshal(responseData, cidResponse)
	if err != nil {
		return nil, err
	}

	return cidResponse, nil
}

// requestBackupFile sends the CID to the network to retrieve the backed up
// file.
func requestBackupFile(cid *cidResponse) ([]byte, error) {

	// Construct restore GET request
	req, err := http.NewRequest(http.MethodGet, recoveryUrl+cid.Value, nil)
	if err != nil {
		return nil, err
	}

	// Add header
	req.Header.Add(basicAuthHeader, recoveryAuthToken)

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
