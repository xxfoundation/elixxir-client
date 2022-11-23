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
	parseFormErr = "Failed to initialize request: %v"
)

// Backup/Pinning constants.
const (
	// URLS
	backupUploadURL = "https://gw-nft.crustapps.net/api/v0/add"
	pinnerURL       = "https://pin.crustcode.com/psa/pins"

	// HTTP POSTing constants
	contentTypeHeader = "Content-Type"
	jsonHeader        = "application/json"
)

////////////////////////////////////////////////////////////////////////////////
// Uploading Backup Logic                                                     //
////////////////////////////////////////////////////////////////////////////////

// UploadSuccessReport is the response given when calling requestPin.
type UploadSuccessReport struct {
	// RequestId is the server returns to the user.
	RequestId string

	// Status is the status of the requestPin received from the server.
	Status string

	// Created is the timestamp that the pin was created.
	Created time.Time
}

// UploadBackup will upload the file provided to the distributed file server.
// This will return a UploadSuccessReport, which provides data on the status of
// the upload. The file may be recovered using RecoverBackup.
func UploadBackup(file BackupFile, privateKey *rsa.PrivateKey,
	udMan *ud.Manager) (*UploadSuccessReport, error) {

	header, err := constructUploadHeader(file, privateKey, udMan)
	if err != nil {
		return nil, errors.Errorf("failed to construct upload header: %+v", err)
	}

	jww.INFO.Printf("[CRUST] Uploading backup file to Crust...")

	// Send backup file to network
	requestBackupResponse, err := uploadBackup(file, header)
	if err != nil {
		return nil, errors.Errorf("failed to upload backup: %+v", err)
	}

	jww.INFO.Printf("[CRUST] Completed upload to Crust.")
	jww.INFO.Printf("[CRUST] Requesting PIN from Crust...")

	// Check on the status of the backup
	uploadSuccess, err := requestPin(requestBackupResponse, header)
	if err != nil {
		return nil, errors.Errorf("failed to request PIN: %+v", err)
	}

	jww.INFO.Printf("[CRUST] Completed PIN request.")

	return uploadSuccess, nil
}

// uploadBackup is a sender function which sends the backup file
// to a backup gateway.
func uploadBackup(file BackupFile, header uploadBackupHeader) (
	*uploadBackupResponse, error) {

	req, err := constructUploadRequest(file, header)
	if err != nil {
		return nil, errors.Errorf("failed to construct request: %+v", err)
	}

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, errors.Errorf("failed request: %+v", err)
	}

	// Handle valid response
	uploadResponse := &uploadBackupResponse{}
	jww.INFO.Printf("[CRUST] responseData %s", string(responseData))
	err = json.Unmarshal(responseData, uploadResponse)
	if err != nil {
		return nil, errors.Errorf("failed to parse response: %+v", err)
	}

	return uploadResponse, nil
}

// requestPin pins the backup to the network.
func requestPin(backupResponse *uploadBackupResponse,
	header uploadBackupHeader) (*UploadSuccessReport, error) {

	// Marshal backup response
	backupJson, err := json.Marshal(backupResponse)
	if err != nil {
		return nil, err
	}

	// Construct pin request
	req, err := http.NewRequest(http.MethodPost, pinnerURL,
		bytes.NewBuffer(backupJson))
	if err != nil {
		return nil, err
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	// Add auth header
	req.SetBasicAuth(header.constructBasicAuth())

	// Add JSON content type header
	req.Header.Add(contentTypeHeader, jsonHeader)

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, errors.Errorf("failed request: %+v", err)
	}

	// Unmarshal response
	uploadSuccess := &UploadSuccessReport{}
	err = json.Unmarshal(responseData, uploadSuccess)
	if err != nil {
		return nil, err
	}

	return uploadSuccess, nil
}
