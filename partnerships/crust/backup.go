////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"net/http"
	"time"
)

// Error constantgitlab.com/xx_network/crypto/tlss
const (
	parseFormErr   = "failed to initialize request: %+v"
	parseRespErr   = "failed to parse response: %+v"
	sendRequestErr = "failed request: %+v"
)

// Backup/Pinning constants.
const (
	// URLS
	backupUploadURL = "https://crustipfs.xyz/api/v0/add"
	pinnerURL       = "https://pin.crustcode.com/psa/pins"

	// HTTP POSTing constants
	contentTypeHeader = "Content-Type"
	jsonHeader        = "application/json; charset=UTF-8"
)

////////////////////////////////////////////////////////////////////////////////
// Uploading Backup Logic                                                     //
////////////////////////////////////////////////////////////////////////////////

type UploadSuccessReport struct {
	// RequestId is the server returns to the user.
	Requestid string `json:"requestid"`
	// Status is the status of the requestPin received from the server.
	Status string `json:"status"`
	// Created is the timestamp that the pin was created.
	Created time.Time `json:"created"`
	Pin     struct {
		Cid     string        `json:"cid"`
		Name    string        `json:"name"`
		Origins []interface{} `json:"origins"`
	} `json:"pin"`
	Delegates []string `json:"delegates"`
	Info      struct {
	} `json:"info"`
}

// UploadBackup will upload the file provided to the distributed file server.
// This will return a UploadSuccessReport, which provides data on the status of
// the upload. The file may be recovered using RecoverBackup.
func UploadBackup(file BackupFile, privateKey *rsa.PrivateKey,
	udMan *ud.Manager) (*UploadSuccessReport, error) {

	jww.INFO.Printf("[CRUST] Backing up file...")

	uploadAuth, err := newUploadAuth(file, privateKey, udMan)
	if err != nil {
		return nil, errors.Errorf("failed to construct upload uploadAuth: %+v", err)
	}

	// Send backup file to network
	requestBackupResponse, err := uploadBackup(file, uploadAuth)
	if err != nil {
		return nil, errors.Errorf("failed to upload backup: %+v", err)
	}

	// Check on the status of the backup
	uploadSuccess, err := requestPin(requestBackupResponse, uploadAuth)
	if err != nil {
		return nil, errors.Errorf("failed to request PIN: %+v", err)
	}

	return uploadSuccess, nil
}

// uploadBackup is a sender function which sends the backup file
// to a backup gateway.
func uploadBackup(file BackupFile, uploadAuth uploadAuth) (
	*uploadBackupResponse, error) {

	jww.DEBUG.Printf("[CRUST] Uploading backup file...")

	req, err := newUploadRequest(file, uploadAuth)
	if err != nil {
		return nil, errors.Errorf("failed to construct request: %+v", err)
	}

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, errors.Errorf(sendRequestErr, err)
	}

	// Handle valid response
	uploadResponse := &uploadBackupResponse{}
	err = json.Unmarshal(responseData, uploadResponse)
	if err != nil {
		return nil, errors.Errorf(parseRespErr, err)
	}

	jww.DEBUG.Printf("[CRUST] Completed upload.")

	return uploadResponse, nil
}

// requestPin pins the backup to the network.
func requestPin(backupResponse *uploadBackupResponse,
	uploadAuth uploadAuth) (*UploadSuccessReport, error) {

	jww.DEBUG.Printf("[CRUST] Requesting PIN...")

	// Construct the pin request
	pinReq := pinRequest{
		Name: backupResponse.Name,
		Cid:  backupResponse.Hash,
	}

	// Write pin into JSON for HTTP request
	jsonData, err := json.Marshal(pinReq)
	if err != nil {
		return nil, err
	}

	// Construct pin request
	req, err := http.NewRequest(http.MethodPost, pinnerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	// Add JSON content type header
	req.Header.Set(contentTypeHeader, jsonHeader)

	// Add get header
	req.SetBasicAuth(uploadAuth.get())

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, errors.Errorf(sendRequestErr, err)
	}

	// Unmarshal response
	uploadSuccess := &UploadSuccessReport{}
	err = json.Unmarshal(responseData, uploadSuccess)
	if err != nil {
		return nil, errors.Errorf(parseRespErr, err)
	}

	return uploadSuccess, nil
}
