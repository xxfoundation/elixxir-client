////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	ft "gitlab.com/elixxir/client/fileTransfer"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"time"
)

// Wrapper handles the sending and receiving of file transfers using the
// Connection interface messages to inform the recipient of incoming file
// transfers.
type Wrapper struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer Manager
	ft ft.FileTransfer

	// Params for wrapper
	p Params

	cmix ft.Cmix
	conn connection
}

// connection interface matches a subset of the connect.Connection methods used
// by the Wrapper for easier testing.
type connection interface {
	GetPartner() partner.Manager
	SendE2E(mt catalog.MessageType, payload []byte, params e2e.Params) (
		cryptoE2e.SendReport, error)
	RegisterListener(messageType catalog.MessageType,
		newListener receive.Listener) (receive.ListenerID, error)
}

// NewWrapper generates a new file transfer manager using connection E2E.
func NewWrapper(receiveCB ft.ReceiveCallback, p Params, ft ft.FileTransfer,
	conn connection, cmix ft.Cmix) (*Wrapper, error) {
	w := &Wrapper{
		receiveCB: receiveCB,
		ft:        ft,
		p:         p,
		cmix:      cmix,
		conn:      conn,
	}

	// Register listener to receive new file transfers
	_, err := w.conn.RegisterListener(catalog.NewFileTransfer, &listener{w})
	if err != nil {
		return nil, err
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

// Send initiates the sending of a file to the connection partner and returns a
// transfer ID that uniquely identifies this file transfer. The initial and
// final messages are sent via connection E2E.
func (w *Wrapper) Send(fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	sendNew := func(transferInfo []byte) error {
		return sendNewFileTransferMessage(transferInfo, w.conn)
	}

	modifiedProgressCB := w.addEndMessageToCallback(progressCB)

	recipient := w.conn.GetPartner().PartnerId()

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

// addEndMessageToCallback adds the sending of a connection E2E message when
// the transfer completed to the callback. If NotifyUponCompletion is not set,
// then the message is not sent.
func (w *Wrapper) addEndMessageToCallback(progressCB ft.SentProgressCallback) ft.SentProgressCallback {
	if !w.p.NotifyUponCompletion {
		return progressCB
	}
	return func(completed bool, arrived, total uint16,
		st ft.SentTransfer, t ft.FilePartTracker, err error) {

		// If the transfer is completed, send last message informing recipient
		if completed {
			sendEndFileTransferMessage(w.cmix, w.conn)
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
