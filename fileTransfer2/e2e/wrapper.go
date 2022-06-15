////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Wrapper handles the sending and receiving of file transfers using E2E
// messages to inform the recipient of incoming file transfers.
type Wrapper struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer Manager
	ft ft.FileTransfer

	// Params for wrapper
	p Params

	myID *id.ID
	cmix ft.Cmix
	e2e  E2e
}

// E2e interface matches a subset of the e2e.Handler methods used by the Wrapper
// for easier testing.
type E2e interface {
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params e2e.Params) ([]id.Round, e2eCrypto.MessageID, time.Time, error)
	RegisterListener(senderID *id.ID, messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
}

// NewWrapper generates a new file transfer manager using E2E.
func NewWrapper(receiveCB ft.ReceiveCallback, p Params, ft ft.FileTransfer,
	myID *id.ID, e2e E2e, cmix ft.Cmix) (*Wrapper, error) {
	w := &Wrapper{
		receiveCB: receiveCB,
		ft:        ft,
		p:         p,
		myID:      myID,
		cmix:      cmix,
		e2e:       e2e,
	}

	// Register listener to receive new file transfers
	w.e2e.RegisterListener(&id.ZeroUser, catalog.NewFileTransfer, &listener{w})

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

// Send initiates the sending of a file to a recipient and returns a transfer ID
// that uniquely identifies this file transfer. The initial and final messages
// are sent via E2E.
func (w *Wrapper) Send(recipient *id.ID, fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	sendNew := func(transferInfo []byte) error {
		return sendNewFileTransferMessage(recipient, transferInfo, w.e2e)
	}

	modifiedProgressCB := w.addEndMessageToCallback(progressCB)

	return w.ft.Send(recipient, fileName, fileType, fileData, retry, preview,
		modifiedProgressCB, period, sendNew)
}

// RegisterSentProgressCallback allows for the registration of a callback to
// track the progress of an individual sent file transfer.
func (w *Wrapper) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {

	modifiedProgressCB := w.addEndMessageToCallback(progressCB)

	return w.ft.RegisterSentProgressCallback(tid, modifiedProgressCB, period)
}

// addEndMessageToCallback adds the sending of an E2E message when the transfer
// completed to the callback. If NotifyUponCompletion is not set, then the
// message is not sent.
func (w *Wrapper) addEndMessageToCallback(
	progressCB ft.SentProgressCallback) ft.SentProgressCallback {
	if !w.p.NotifyUponCompletion {
		return progressCB
	}
	return func(completed bool, arrived, total uint16,
		st ft.SentTransfer, t ft.FilePartTracker, err error) {

		// If the transfer is completed, send last message informing recipient
		if completed {
			sendEndFileTransferMessage(st.Recipient(), w.cmix, w.e2e)
		}

		progressCB(completed, arrived, total, st, t, err)
	}
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
