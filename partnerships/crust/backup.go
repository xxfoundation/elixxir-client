////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/netTime"
	"net/http"
	"strconv"
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
	fileKey           = "file"
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

// uploadBackupHeader is the header that will be sent to the
// Client's connection using Client.UploadChatHistory.
type uploadBackupHeader struct {

	// UserPublicKey is the user's public key PEM encoded.
	UserPublicKey []byte

	// Username is the user's username.
	Username string

	// VerificationSignature is the signature indicating that this owner
	// owns their username. This is obtained via [ud.Manager]'s
	// GetUsernameValidationSignature method.
	VerificationSignature []byte

	// UploadSignature is the signature of the file being uploaded.
	// This may be generated using [crust.SignUpload].
	UploadSignature []byte

	// UploadTimestamp is the timestamp in which the user wanted to upload
	// the file. This is what's passed into [crust.SignUpload].
	UploadTimestamp int64

	// FileHash is the hash of the file to be backed up. This can be obtained
	// using [crust.HashFile].
	FileHash []byte
}

// uploadBackupResponse is the response received from uploadBackup
// after sending a backup file and a uploadBackupHeader.
type uploadBackupResponse struct {
	Name string

	// Hash is the CID returned when uploading a backup.
	Hash string

	// The size of the file.
	Size int
}

// UploadBackup will upload the file provided to the distributed file server.
// This will return a UploadSuccessReport, which provides data on the status of
// the upload. The file may be recovered using RecoverBackup.
func UploadBackup(file []byte, privateKey *rsa.PrivateKey,
	udMan *ud.Manager) (*UploadSuccessReport, error) {

	// Retrieve validation signature
	verificationSignature, err := udMan.GetUsernameValidationSignature()
	if err != nil {
		return nil, errors.Errorf("failed to get username "+
			"validation signature: %+v", err)
	}

	// Retrieve username
	username, err := udMan.GetUsername()
	if err != nil {
		return nil, errors.Errorf("failed to get username: %+v", err)
	}

	// Hash the file
	fileHash, err := crust.HashFile(file)
	if err != nil {
		return nil, errors.Errorf("failed to hash file: %+v", err)
	}

	// Sign the upload
	uploadTimestamp := netTime.Now()
	uploadSignature, err := crust.SignUpload(rand.Reader,
		privateKey, file, uploadTimestamp)
	if err != nil {
		return nil, errors.Errorf("failed to sign upload: %+v", err)
	}

	// Serialize the public key PEM
	pubKeyPem := rsa.CreatePublicKeyPem(privateKey.GetPublic())

	// Construct header
	header := uploadBackupHeader{
		UserPublicKey:         pubKeyPem,
		Username:              username,
		VerificationSignature: verificationSignature,
		UploadSignature:       uploadSignature,
		UploadTimestamp:       uploadTimestamp.UnixNano(),
		FileHash:              fileHash,
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
func uploadBackup(file []byte, header uploadBackupHeader) (
	*uploadBackupResponse, error) {

	// Construct upload POST request
	req, err := http.NewRequest(http.MethodPost, backupUploadURL, http.NoBody)
	if err != nil {
		return nil, errors.Errorf("Failed to construct request: %v", err)
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	// Add file
	req.Form.Add(fileKey, string(file))

	// Add auth header
	req.SetBasicAuth(header.constructBasicAuth())

	// Send request
	responseData, err := sendRequest(req)
	if err != nil {
		return nil, errors.Errorf("Failed to send request: %+v", err)
	}

	// Handle valid response
	uploadResponse := &uploadBackupResponse{}
	err = json.Unmarshal(responseData, uploadResponse)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// Unmarshal response
	uploadSuccess := &UploadSuccessReport{}
	err = json.Unmarshal(responseData, uploadSuccess)
	if err != nil {
		return nil, err
	}

	return uploadSuccess, nil
}

// constructBasicAuth is a helper function which constructs
// the header into a username:password format for the http.Request's
// BasicAuth function.
func (header uploadBackupHeader) constructBasicAuth() (
	username, password string) {
	username = fmt.Sprintf("xx-%s-%s-%s-%s-%s",
		base64.StdEncoding.EncodeToString(header.UserPublicKey),
		header.Username,
		base64.StdEncoding.EncodeToString(header.FileHash),
		strconv.FormatInt(header.UploadTimestamp, 10),
		base64.StdEncoding.EncodeToString(header.UploadSignature),
	)

	password = fmt.Sprintf("%s",
		base64.StdEncoding.EncodeToString(header.VerificationSignature),
	)

	return
}
