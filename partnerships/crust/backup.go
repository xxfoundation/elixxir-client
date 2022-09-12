////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/netTime"
	"net/http"
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

// uploadBackupHeader is the header that will be sent to the
// Client's connection using Client.UploadChatHistory.
type uploadBackupHeader struct {

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
	UploadTimestamp int64

	// FileHash is the hash of the file to be backed up. This can be obtained
	// using [crust.HashFile].
	FileHash []byte
}

// serialize is a helper function which serializes the header as per spec.
func (header uploadBackupHeader) serialize() string {
	auth := []byte(fmt.Sprintf("xx-%s-%s-%s-%d-%s:%s",
		header.UserPublicKey,
		base64.StdEncoding.EncodeToString(header.UsernameHash),
		base64.StdEncoding.EncodeToString(header.FileHash),
		header.UploadTimestamp,
		base64.StdEncoding.EncodeToString(header.UploadSignature),
		base64.StdEncoding.EncodeToString(header.VerificationSignature),
	))

	return base64.StdEncoding.EncodeToString(auth)
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
// This will return a pinResponse, which provides data on the status of the
// upload. The file may be recovered using RecoverBackup.
func UploadBackup(file []byte, privateKey *rsa.PrivateKey,
	udMan *ud.Manager) error {

	// Retrieve validation signature
	verificationSignature, err := udMan.GetUsernameValidationSignature()
	if err != nil {
		return errors.Errorf("failed to get username "+
			"validation signature: %+v", err)
	}

	// Retrieve username
	username, err := udMan.GetUsername()
	if err != nil {
		return errors.Errorf("failed to get username: %+v", err)
	}

	// Hash the username
	usernameHash := crust.HashUsername(username)

	// Hash the file
	fileHash, err := crust.HashFile(file)
	if err != nil {
		return errors.Errorf("failed to hash file: %+v", err)
	}

	// Sign the upload
	uploadTimestamp := netTime.Now()
	uploadSignature, err := crust.SignUpload(rand.Reader,
		privateKey, file, uploadTimestamp)
	if err != nil {
		return errors.Errorf("failed to sign upload: %+v", err)
	}

	// Serialize the public key PEM
	pubKeyPem := string(rsa.CreatePublicKeyPem(privateKey.GetPublic()))

	// Construct header
	header := uploadBackupHeader{
		UserPublicKey:         pubKeyPem,
		UsernameHash:          usernameHash,
		VerificationSignature: verificationSignature,
		UploadSignature:       uploadSignature,
		UploadTimestamp:       uploadTimestamp.UnixNano(),
		FileHash:              fileHash,
	}

	// Send backup file to network
	requestBackupResponse, err := uploadBackup(file, header.serialize())
	if err != nil {
		return errors.Errorf("failed to upload backup: %+v", err)
	}

	// Check on the status of the backup
	err = requestPin(requestBackupResponse, header.serialize())
	if err != nil {
		return errors.Errorf("failed to request PIN: %+v", err)
	}

	return nil
}

// uploadBackup is a sender function which sends the backup file
// to a backup gateway.
func uploadBackup(file []byte, serializedHeaderInfo string) (
	*uploadBackupResponse, error) {

	// Construct upload POST request
	req, err := http.NewRequest(http.MethodPost, backupUploadURL, nil)
	if err != nil {
		return nil, err
	}

	// Add file
	req.Form.Add(fileKey, string(file))

	// Add header
	req.Header.Add(basicAuthHeader, serializedHeaderInfo)

	responseData, err := sendRequest(req)
	if err != nil {
		return nil, err
	}

	// Handle valid response
	uploadResponse := &uploadBackupResponse{}
	err = json.Unmarshal(responseData, uploadResponse)
	if err != nil {
		return nil, err
	}

	return uploadResponse, nil
}

////////////////////////////////////////////////////////////////////////////////
// Pinning Backup Logic                                                       //
////////////////////////////////////////////////////////////////////////////////

// pinResponse is the response given when calling requestPin.
type pinResponse struct {
	// RequestId is the server returns to the user.
	RequestId string

	// Status is the status of the requestPin received from the server.
	Status string

	// Created is the timestamp that the pin was created.
	Created time.Time
}

// requestPin pins the backup to the network.
func requestPin(backupResponse *uploadBackupResponse,
	serializedHeader string) error {

	// Construct pin request
	req, err := http.NewRequest(http.MethodPost, pinnerURL, nil)
	if err != nil {
		return err
	}

	// Marshal backup response
	backupJson, err := json.Marshal(backupResponse)
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Add(basicAuthHeader, serializedHeader)
	req.Header.Add(jsonHeader, string(backupJson))

	// Send request
	_, err = sendRequest(req)
	if err != nil {
		return err
	}

	return nil
}
