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
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Server URLs for backing up.
const (
	backupUploadURL = "https://crustipfs.xyz/api/v0/add"
	pinnerURL       = "https://pin.crustcode.com/psa/pins"
)

// HTTP POST headers relevant for backing up.
const (
	basicAuthHeader = "Authorization: Basic"
	jsonHeader      = "Content-Type: application/json"
	fileKey         = "file"
)

////////////////////////////////////////////////////////////////////////////////
// Uploading Backup Logic                                                     //
////////////////////////////////////////////////////////////////////////////////

// UploadBackupHeader is the header that will be sent to the
// Client's connection using Client.UploadChatHistory.
type UploadBackupHeader struct {

	// UserPublicKey is the user's public key PEM encoded.
	UserPublicKey string

	// UsernameHash is the hash of the user's username. This can be obtained
	//	// using [crust.HashUsername].
	UsernameHash []byte

	// VerificationSignature is the signature indicating that this owner
	// owns their username. This is obtained via [ud.Manager]'s
	// GetUsernameValidationSignature method.
	VerificationSignature []byte

	// UploadSignature is the signature of the file being uploaded.
	// This may be generated using [crust.SignUpload].
	UploadSignature []byte

	// UploadTimestamp is the timestamp in which the user wanted to upload
	// the file. This is what's passed into [crust.SignUpload].
	UploadTimestamp int

	// FileHash is the hash of the file to be backed up. This can be obtained
	// using [crust.HashFile].
	FileHash []byte
}

// serialize is a helper function which serializes the header as per spec.
func (header UploadBackupHeader) serialize() string {
	auth := []byte(fmt.Sprintf("xx-%s-%s-%s-%s-%s:%s",
		header.UserPublicKey,
		base64.StdEncoding.EncodeToString(header.UsernameHash),
		base64.StdEncoding.EncodeToString(header.FileHash),
		strconv.Itoa(header.UploadTimestamp),
		base64.StdEncoding.EncodeToString(header.UploadSignature),
		base64.StdEncoding.EncodeToString(header.VerificationSignature),
	))

	return base64.StdEncoding.EncodeToString(auth)
}

// UploadBackupResponse is the response received from RequestUploadBackup
// after sending a backup file and a UploadBackupHeader.
type UploadBackupResponse struct {
	Name string

	// Hash is the CID returned when uploading a backup.
	Hash string

	// The size of the file.
	Size int
}

// RequestUploadBackup is a sender function which sends the backup file
// to a backup gateway..
func RequestUploadBackup(file string, headerInfo UploadBackupHeader) (
	*UploadBackupResponse, error) {

	// Construct upload POST request
	req, err := http.NewRequest(http.MethodPost, backupUploadURL, nil)
	if err != nil {
		return nil, err
	}

	// Add file
	req.Form.Add(fileKey, file)

	// Add header
	req.Header.Add(basicAuthHeader, headerInfo.serialize())

	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Handle valid response
	uploadResponse := &UploadBackupResponse{}
	err = json.Unmarshal(responseData, uploadResponse)
	if err != nil {
		return nil, err
	}

	return uploadResponse, nil
}

////////////////////////////////////////////////////////////////////////////////
// Pinning Backup Logic                                                       //
////////////////////////////////////////////////////////////////////////////////

// PinResponse is the response given when calling RequestPin.
type PinResponse struct {
	// RequestId is the server returns to the user.
	RequestId string

	// Status is the status of the RequestPin received from the server.
	Status string

	// Created is the timestamp that the pin was created.
	Created time.Time
}

// RequestPin pins the backup to the network.
func RequestPin(headerInfo UploadBackupHeader,
	backupResponse *UploadBackupResponse) (*PinResponse, error) {

	// Construct pin request
	req, err := http.NewRequest(http.MethodPost, pinnerURL, nil)
	if err != nil {
		return nil, err
	}

	// Marshal backup response
	backupJson, err := json.Marshal(backupResponse)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Add(basicAuthHeader, headerInfo.serialize())
	req.Header.Add(jsonHeader, string(backupJson))

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Unmarshal response
	pinResponse := &PinResponse{}
	err = json.Unmarshal(responseData, pinResponse)
	if err != nil {
		return nil, err
	}

	return pinResponse, nil
}
