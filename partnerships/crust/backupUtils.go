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
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/netTime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
)

// uploadBackupAuth is the header that will be used for posting a backup
// to Crust's architecture.
type uploadBackupAuth struct {

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
// after sending a backup file and a uploadBackupAuth.
type uploadBackupResponse struct {
	Name string

	// Hash is the CID returned when uploading a backup.
	Hash string

	// The size of the file.
	Size string
}

// pinRequest is the request struct used in requestPin.
type pinRequest struct {
	Name string `json:"name"`
	Cid  string `json:"cid"`
}

// newUploadAuth is a constructor for the uploadBackupAuth.
// This is used to create a
func newUploadAuth(file BackupFile, privateKey *rsa.PrivateKey,
	udMan *ud.Manager) (uploadBackupAuth, error) {

	// Retrieve validation signature
	verificationSignature, err := udMan.GetUsernameValidationSignature()
	if err != nil {
		return uploadBackupAuth{},
			errors.Errorf("failed to get username validation signature: %+v", err)
	}

	// Retrieve username
	username, err := udMan.GetUsername()
	if err != nil {
		return uploadBackupAuth{}, errors.Errorf("failed to get username: %+v", err)
	}

	// Hash the file
	fileHash, err := crust.HashFile(file.Data)
	if err != nil {
		return uploadBackupAuth{}, errors.Errorf("failed to hash file: %+v", err)
	}

	// Sign the upload
	uploadTimestamp := netTime.Now()
	uploadSignature, err := crust.SignUpload(rand.Reader,
		privateKey, file.Data, uploadTimestamp)
	if err != nil {
		return uploadBackupAuth{}, errors.Errorf("failed to sign upload: %+v", err)
	}

	// Serialize the public key PEM
	pubKeyPem := rsa.CreatePublicKeyPem(privateKey.GetPublic())

	// Construct header
	header := uploadBackupAuth{
		UserPublicKey:         pubKeyPem,
		Username:              username,
		VerificationSignature: verificationSignature,
		UploadSignature:       uploadSignature,
		UploadTimestamp:       uploadTimestamp.UnixNano(),
		FileHash:              fileHash,
	}

	return header, nil
}

// getAuth is a helper function which constructs the HTTP basic auth information
// from uploadBackupAuth. This returns it as a username-password pair such that
// it can be passed directly into the http.Request object's SetBasicAuth method.
func (header uploadBackupAuth) getAuth() (username, password string) {
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

// constructUploadRequest is a helper function which constructs a http.Request
// for uploading a backup file.
func constructUploadRequest(file BackupFile, uploadAuth uploadBackupAuth) (
	*http.Request, error) {
	// Serialize file into body
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	part, err := writer.CreateFormFile("file",
		filepath.Base(file.Path))
	if err != nil {
		return nil, err
	}
	_, err = part.Write(file.Data)
	if err != nil {
		return nil, err
	}

	if err = writer.Close(); err != nil {
		return nil, err
	}

	// Construct upload POST request
	req, err := http.NewRequest(http.MethodPost, backupUploadURL, buf)
	if err != nil {
		return nil, errors.Errorf("failed to construct request: %v", err)
	}

	// Initialize request to fill out Form section
	err = req.ParseForm()
	if err != nil {
		return nil, errors.Errorf(parseFormErr, err)
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	// Add auth header
	req.SetBasicAuth(uploadAuth.getAuth())

	return req, nil
}
