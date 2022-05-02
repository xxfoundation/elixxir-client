////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Error messages.
const (
	// NewManager
	errNewFtManager = "cannot create new E2E file transfer manager: %+v"
)

// manager handles the sending and receiving of file transfers using E2E
// messages to inform the recipient of incoming file transfers.
type manager struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer manager
	ft ft.FileTransfer

	myID *id.ID
	cmix ft.Cmix
	e2e  E2e
}

// E2e interface matches a subset of the e2e.Handler methods used by the manager
// for easier testing.
type E2e interface {
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params e2e.Params) ([]id.Round, e2eCrypto.MessageID, time.Time, error)
	RegisterListener(senderID *id.ID, messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID
}

// NewManager generates a new file transfer manager using E2E.
func NewManager(receiveCB ft.ReceiveCallback, params ft.Params, myID *id.ID,
	e2e E2e, cmix ft.Cmix, kv *versioned.KV, rng *fastRNG.StreamGenerator) (
	ft.FileTransfer, error) {

	sendNewCb := func(recipient *id.ID, info *ft.TransferInfo) error {
		return sendNewFileTransferMessage(recipient, info, e2e)
	}

	sendEndCb := func(recipient *id.ID) {
		sendEndFileTransferMessage(recipient, cmix, e2e)
	}

	ftManager, err := ft.NewManager(
		sendNewCb, sendEndCb, params, myID, cmix, kv, rng)
	if err != nil {
		return nil, errors.Errorf(errNewFtManager, err)
	}

	return &manager{
		receiveCB: receiveCB,
		ft:        ftManager,
		myID:      myID,
		cmix:      cmix,
		e2e:       e2e,
	}, nil
}

func (m *manager) StartProcesses() (stoppable.Stoppable, error) {
	// Register listener to receive new file transfers
	m.e2e.RegisterListener(m.myID, catalog.NewFileTransfer, &listener{m})

	return m.ft.StartProcesses()
}

func (m *manager) MaxFileNameLen() int {
	return m.ft.MaxFileNameLen()
}

func (m *manager) MaxFileTypeLen() int {
	return m.ft.MaxFileTypeLen()
}

func (m *manager) MaxFileSize() int {
	return m.ft.MaxFileSize()
}

func (m *manager) MaxPreviewSize() int {
	return m.ft.MaxPreviewSize()
}

func (m *manager) Send(fileName, fileType string, fileData []byte,
	recipient *id.ID, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {
	return m.ft.Send(fileName, fileType, fileData, recipient, retry, preview,
		progressCB, period)
}

func (m *manager) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {
	return m.ft.RegisterSentProgressCallback(tid, progressCB, period)
}

func (m *manager) CloseSend(tid *ftCrypto.TransferID) error {
	return m.ft.CloseSend(tid)
}

func (m *manager) AddNew(fileName string, key *ftCrypto.TransferKey,
	transferMAC []byte, numParts uint16, size uint32, retry float32,
	progressCB ft.ReceivedProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {

	return m.ft.AddNew(fileName, key, transferMAC, numParts, size, retry,
		progressCB, period)
}

func (m *manager) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.ReceivedProgressCallback, period time.Duration) error {
	return m.ft.RegisterReceivedProgressCallback(tid, progressCB, period)
}

func (m *manager) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	return m.ft.Receive(tid)
}
