////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package distributedBackup

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
)

// Recovery URLs and tokens.
const (
	cidRequestURL     = "https://pin.crustcode.com/psa/value?key="
	recoveryUrl       = "https://crustipfs.xyz/ipfs/"
	recoveryAuthToken = `c3ViLTVGcXdqWW9MUXE5Z2NXSGM2azVZd3RuZHg3Q01pQ1FlS1N2THpYb2plZ3ZYY005bToweDIwZjYxODc5ODhiNjIxZjk3ZDQwNGZlZmMzZTQ5MWU5OTU0MWY5MzdjNDNiYWQyNWE3NGQ2OTRjMjA2NWYzODFkOTc5YmY0YTQzYmI3MGZlYzY1MzIwMGI2MmFhZmRiOWFjNzEwODc1YzlkMjhlYTJjNTA2ZDcyZDc1Y2RjNzA2`
)

// CidResponse is the response from the RequestCid call to the pinner.
type CidResponse struct {
	// Value is the CID associated with our username. This
	// value allows us to request the file from the network.
	Value string
}

// RecoveryResponse is the response from Restore.
type RecoveryResponse struct {
	// Value is the base64 encoded file that was backup up to the network.
	Value string
}

// RequestCid requests the CID associated with this username.
// This allows the user to request their backup file using Recover.
func RequestCid(usernameHash string) (*CidResponse, error) {

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
	cidResponse := &CidResponse{}
	err = json.Unmarshal(responseData, cidResponse)
	if err != nil {
		return nil, err
	}

	return cidResponse, nil
}

// Recover sends the CID to the network to retrieve the backed up file.
func Recover(cid *CidResponse) ([]byte, error) {

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
