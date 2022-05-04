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
	"gitlab.com/elixxir/client/stoppable"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Manager handles the sending and receiving of file transfers using E2E
// messages to inform the recipient of incoming file transfers.
type Manager struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer Manager
	ft ft.FileTransfer

	myID *id.ID
	cmix ft.Cmix
	e2e  E2e
}

// E2e interface matches a subset of the e2e.Handler methods used by the Manager
// for easier testing.
type E2e interface {
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params e2e.Params) ([]id.Round, e2eCrypto.MessageID, time.Time, error)
	RegisterListener(senderID *id.ID, messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
}

// NewManager generates a new file transfer manager using E2E.
func NewManager(receiveCB ft.ReceiveCallback, ft ft.FileTransfer, myID *id.ID,
	e2e E2e, cmix ft.Cmix) (*Manager, error) {
	return &Manager{
		receiveCB: receiveCB,
		ft:        ft,
		myID:      myID,
		cmix:      cmix,
		e2e:       e2e,
	}, nil
}

func (m *Manager) StartProcesses() (stoppable.Stoppable, error) {
	// Register listener to receive new file transfers
	m.e2e.RegisterListener(m.myID, catalog.NewFileTransfer, &listener{m})

	return m.ft.StartProcesses()
}

func (m *Manager) MaxFileNameLen() int {
	return m.ft.MaxFileNameLen()
}

func (m *Manager) MaxFileTypeLen() int {
	return m.ft.MaxFileTypeLen()
}

func (m *Manager) MaxFileSize() int {
	return m.ft.MaxFileSize()
}

func (m *Manager) MaxPreviewSize() int {
	return m.ft.MaxPreviewSize()
}

func (m *Manager) Send(recipient *id.ID, fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	sendNew := func(info *ft.TransferInfo) error {
		return sendNewFileTransferMessage(recipient, info, m.e2e)
	}

	modifiedProgressCB := m.addEndMessageToCallback(progressCB)

	return m.ft.Send(recipient, fileName, fileType, fileData, retry, preview,
		modifiedProgressCB, period, sendNew)
}

func (m *Manager) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {

	modifiedProgressCB := m.addEndMessageToCallback(progressCB)

	return m.ft.RegisterSentProgressCallback(tid, modifiedProgressCB, period)
}

func (m *Manager) addEndMessageToCallback(progressCB ft.SentProgressCallback) ft.SentProgressCallback {
	return func(completed bool, arrived, total uint16,
		st ft.SentTransfer, t ft.FilePartTracker, err error) {

		// If the transfer is completed, send last message informing recipient
		if completed {
			sendEndFileTransferMessage(st.Recipient(), m.cmix, m.e2e)
		}

		progressCB(completed, arrived, total, st, t, err)
	}
}

func (m *Manager) CloseSend(tid *ftCrypto.TransferID) error {
	return m.ft.CloseSend(tid)
}

func (m *Manager) HandleIncomingTransfer(fileName string,
	key *ftCrypto.TransferKey, transferMAC []byte, numParts uint16, size uint32,
	retry float32, progressCB ft.ReceivedProgressCallback,
	period time.Duration) (*ftCrypto.TransferID, error) {

	return m.ft.HandleIncomingTransfer(
		fileName, key, transferMAC, numParts, size, retry, progressCB, period)
}

func (m *Manager) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.ReceivedProgressCallback, period time.Duration) error {
	return m.ft.RegisterReceivedProgressCallback(tid, progressCB, period)
}

func (m *Manager) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	return m.ft.Receive(tid)
}
