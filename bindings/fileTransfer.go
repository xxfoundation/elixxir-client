////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	ft "gitlab.com/elixxir/client/fileTransfer"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// FileTransfer contains the file transfer manager.
type FileTransfer struct {
	m *ft.Manager
}

// FileTransferSentProgressFunc contains a function callback that tracks the
// progress of sending a file. It is called when a file part is sent, a file
// part arrives, the transfer completes, or on error.
type FileTransferSentProgressFunc interface {
	SentProgressCallback(completed bool, sent, arrived, total int, err error)
}

// FileTransferReceivedProgressFunc contains a function callback that tracks the
// progress of receiving a file. It is called when a file part is received, the
// transfer completes, or on error.
type FileTransferReceivedProgressFunc interface {
	ReceivedProgressCallback(completed bool, received, total int, err error)
}

// FileTransferReceiveFunc contains a function callback that notifies the
// receiver of an incoming file transfer. It is called on the reception of the
// initial file transfer message.
type FileTransferReceiveFunc interface {
	ReceiveCallback(tid []byte, fileName string, sender []byte, size int,
		preview []byte)
}

// NewFileTransferManager creates a new file transfer manager and starts the
// sending and receiving threads. The receiveFunc is called everytime a new file
// transfer is received. The parameters string is a JSON formatted string of the
// fileTransfer.Params object. If it is left empty, then defaults are used. It
// must match the following format: {"MaxThroughput":150000}
func NewFileTransferManager(client *Client, receiveFunc FileTransferReceiveFunc,
	parameters string) (FileTransfer, error) {

	receiveCB := func(tid ftCrypto.TransferID, fileName string, sender *id.ID,
		size uint32, preview []byte) {
		receiveFunc.ReceiveCallback(
			tid.Bytes(), fileName, sender.Bytes(), int(size), preview)
	}

	// JSON unmarshal parameters string
	p := ft.DefaultParams()
	if parameters != "" {
		err := json.Unmarshal([]byte(parameters), &p)
		if err != nil {
			return FileTransfer{}, err
		}
	}

	// Create new file transfer manager
	m, err := ft.NewManager(&client.api, receiveCB, p)
	if err != nil {
		return FileTransfer{}, err
	}

	// Start sending and receiving threads
	err = client.api.AddService(m.StartProcesses)
	if err != nil {
		return FileTransfer{}, err
	}

	return FileTransfer{m}, nil
}

// Send sends a file to the recipient. The sender must have an E2E relationship
// with the recipient.
// The file name is the name of the file to show a user. It has a max length of
// 32 bytes.
// The file data cannot be larger than 4 mB
// The retry float is the total amount of data to send relative to the data
// size. Data will be resent on error and will resend up to [(1 + retry) *
// fileSize].
// The preview stores a preview of the data (such as a thumbnail) and is
// capped at 4 kB in size.
// Returns a unique transfer ID used to identify the transfer.
// PeriodMS is the duration, in milliseconds, to wait between progress callback
// calls. Set this large enough to prevent spamming.
func (f FileTransfer) Send(fileName string, fileData []byte, recipientID []byte,
	retry float32, preview []byte, progressFunc FileTransferSentProgressFunc,
	periodMS int) ([]byte, error) {

	// Create SentProgressCallback
	progressCB := func(completed bool, sent, arrived, total uint16, err error) {
		progressFunc.SentProgressCallback(
			completed, int(sent), int(arrived), int(total), err)
	}

	// Convert recipient ID bytes to id.ID
	recipient, err := id.Unmarshal(recipientID)
	if err != nil {
		return []byte{}, err
	}

	// Convert period to time.Duration
	period := time.Duration(periodMS) * time.Millisecond

	// Send file
	tid, err := f.m.Send(
		fileName, fileData, recipient, retry, preview, progressCB, period)
	if err != nil {
		return nil, err
	}

	// Return transfer ID as bytes and error
	return tid.Bytes(), nil
}

// RegisterSendProgressCallback allows for the registration of a callback to
// track the progress of an individual sent file transfer. The callback will be
// called immediately when added to report the current status of the transfer.
// It will then call every time a file part is sent, a file part arrives, the
// transfer completes, or an error occurs. It is called at most once every
// period, which means if events occur faster than the period, then they will
// not be reported and instead the progress will be reported once at the end of
// the period.
// The period is specified in milliseconds.
func (f FileTransfer) RegisterSendProgressCallback(transferID []byte,
	progressFunc FileTransferSentProgressFunc, periodMS int) error {

	// Unmarshal transfer ID
	tid := ftCrypto.UnmarshalTransferID(transferID)

	// Create SentProgressCallback
	progressCB := func(completed bool, sent, arrived, total uint16, err error) {
		progressFunc.SentProgressCallback(
			completed, int(sent), int(arrived), int(total), err)
	}

	// Convert period to time.Duration
	period := time.Duration(periodMS) * time.Millisecond

	return f.m.RegisterSendProgressCallback(tid, progressCB, period)
}

// Resend resends a file if Send fails.
func (f FileTransfer) Resend(transferID []byte) error {
	// Unmarshal transfer ID
	tid := ftCrypto.UnmarshalTransferID(transferID)

	return f.m.Resend(tid)
}

// CloseSend deletes a sent file transfer from the sent transfer map and from
// storage once a transfer has completed or reached the retry limit. Returns an
// error if the transfer has not run out of retries.
func (f FileTransfer) CloseSend(transferID []byte) error {
	// Unmarshal transfer ID
	tid := ftCrypto.UnmarshalTransferID(transferID)

	return f.m.CloseSend(tid)
}

// RegisterReceiveProgressCallback allows for the registration of a callback to
// track the progress of an individual received file transfer. The callback will
// be called immediately when added to report the current status of the
// transfer. It will then call every time a file part is received, the transfer
// completes, or an error occurs. It is called at most once ever period, which
// means if events occur faster than the period, then they will not be reported
// and instead the progress will be reported once at the end of the period.
// Once the callback reports that the transfer has completed, the recipient
// can get the full file by calling Receive.
// The period is specified in milliseconds.
func (f FileTransfer) RegisterReceiveProgressCallback(transferID []byte,
	progressFunc FileTransferReceivedProgressFunc, periodMS int) error {
	// Unmarshal transfer ID
	tid := ftCrypto.UnmarshalTransferID(transferID)

	// Create ReceivedProgressCallback
	progressCB := func(completed bool, received, total uint16, err error) {
		progressFunc.ReceivedProgressCallback(
			completed, int(received), int(total), err)
	}

	// Convert period to time.Duration
	period := time.Duration(periodMS) * time.Millisecond

	return f.m.RegisterReceiveProgressCallback(tid, progressCB, period)
}

// Receive returns the fully assembled file on the completion of the transfer.
// It deletes the transfer from the received transfer map and from storage.
// Returns an error if the transfer is not complete, the full file cannot be
// verified, or if the transfer cannot be found.
func (f FileTransfer) Receive(transferID []byte) ([]byte, error) {
	// Unmarshal transfer ID
	tid := ftCrypto.UnmarshalTransferID(transferID)

	return f.m.Receive(tid)
}

////////////////////////////////////////////////////////////////////////////////
// Utility Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// GetMaxFilePreviewSize returns the maximum file preview size, in bytes.
func (f FileTransfer) GetMaxFilePreviewSize() int {
	return ft.PreviewMaxSize
}

// GetMaxFileNameByteLength returns the maximum length, in bytes, allowed for a
// file name.
func (f FileTransfer) GetMaxFileNameByteLength() int {
	return ft.FileNameMaxLen
}

// GetMaxFileSize returns the maximum file size, in bytes, allowed to be
// transferred.
func (f FileTransfer) GetMaxFileSize() int {
	return ft.FileMaxSize
}
