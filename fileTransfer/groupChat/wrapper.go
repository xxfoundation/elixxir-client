////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	ft "gitlab.com/elixxir/client/fileTransfer"
	"gitlab.com/elixxir/client/groupChat"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Error messages.
const (
	// Wrapper.StartProcesses
	errAddNewService = "failed to add service to receive new group file transfers: %+v"
)

const (
	// Tag used when sending/receiving new group chat file transfers message
	newFileTransferTag = "NewGroupFileTransfer"
)

// Wrapper handles the sending and receiving of file transfers for group chats.
type Wrapper struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer Manager
	ft ft.FileTransfer

	// Group chat Manager
	gc GroupChat
}

// GroupChat interface matches a subset of the groupChat.GroupChat methods used
// by the Wrapper for easier testing.
type GroupChat interface {
	Send(groupID *id.ID, tag string, message []byte) (
		id.Round, time.Time, group.MessageID, error)
	AddService(tag string, p groupChat.Processor) error
}

// NewWrapper generates a new file transfer Wrapper for group chat.
func NewWrapper(receiveCB ft.ReceiveCallback, ft ft.FileTransfer, gc GroupChat) (
	*Wrapper, error) {
	w := &Wrapper{
		receiveCB: receiveCB,
		ft:        ft,
		gc:        gc,
	}

	err := w.gc.AddService(newFileTransferTag, &processor{w})
	if err != nil {
		return nil, errors.Errorf(errAddNewService, err)
	}

	return w, nil
}

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (w *Wrapper) MaxFileNameLen() int {
	return w.ft.MaxFileNameLen()
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (w *Wrapper) MaxFileTypeLen() int {
	return w.ft.MaxFileTypeLen()
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (w *Wrapper) MaxFileSize() int {
	return w.ft.MaxFileSize()
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (w *Wrapper) MaxPreviewSize() int {
	return w.ft.MaxPreviewSize()
}

// Send initiates the sending of a file to a group and returns a transfer ID
// that uniquely identifies this file transfer.
func (w *Wrapper) Send(groupID *id.ID, fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {
	sendNew := func(transferInfo []byte) error {
		return sendNewFileTransferMessage(groupID, transferInfo, w.gc)
	}

	return w.ft.Send(groupID, fileName, fileType, fileData, retry, preview,
		progressCB, period, sendNew)
}

// RegisterSentProgressCallback allows for the registration of a callback to
// track the progress of an individual sent file transfer.
func (w *Wrapper) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {
	return w.ft.RegisterSentProgressCallback(tid, progressCB, period)
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. Returns an error if the transfer
// has not run out of retries.
//
// This function should be called once a transfer completes or errors out
// (as reported by the progress callback).
func (w *Wrapper) CloseSend(tid *ftCrypto.TransferID) error {
	return w.ft.CloseSend(tid)
}

// RegisterReceivedProgressCallback allows for the registration of a callback to
// track the progress of an individual received file transfer. This must be done
// when a new transfer is received on the ReceiveCallback.
func (w *Wrapper) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.ReceivedProgressCallback, period time.Duration) error {
	return w.ft.RegisterReceivedProgressCallback(tid, progressCB, period)
}

// Receive returns the full file on the completion of the transfer.
// It deletes internal references to the data and unregisters any attached
// progress callback. Returns an error if the transfer is not complete, the
// full file cannot be verified, or if the transfer cannot be found.
//
// Receive can only be called once the progress callback returns that the
// file transfer is complete.
func (w *Wrapper) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	return w.ft.Receive(tid)
}
