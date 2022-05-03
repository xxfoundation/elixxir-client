////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/elixxir/client/groupChat"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Error messages.
const (
	// NewManager
	errNewFtManager = "cannot create new group chat file transfer manager: %+v"

	// manager.StartProcesses
	errAddNewService = "failed to add service to receive new group file transfers: %+v"
)

const (
	// Tag used when sending/receiving new group chat file transfers message
	newFileTransferTag = "NewGroupFileTransfer"

	// Tag used when sending/receiving end group chat file transfers message
	endFileTransferTag = "EndGroupFileTransfer"
)

// manager handles the sending and receiving of file transfers for group chats.
type manager struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer manager
	ft ft.FileTransfer

	// Group chat manager
	gc groupChat.GroupChat

	myID *id.ID
	cmix ft.Cmix
}

// NewManager generates a new file transfer manager for group chat.
func NewManager(receiveCB ft.ReceiveCallback, params ft.Params, myID *id.ID,
	gc groupChat.GroupChat, cmix ft.Cmix, kv *versioned.KV,
	rng *fastRNG.StreamGenerator) (ft.FileTransfer, error) {

	ftManager, err := ft.NewManager(params, myID, cmix, kv, rng)
	if err != nil {
		return nil, errors.Errorf(errNewFtManager, err)
	}

	return &manager{
		receiveCB: receiveCB,
		ft:        ftManager,
		gc:        gc,
		myID:      myID,
		cmix:      cmix,
	}, nil
}

func (m *manager) StartProcesses() (stoppable.Stoppable, error) {
	err := m.gc.AddService(newFileTransferTag, &processor{m})
	if err != nil {
		return nil, errors.Errorf(errAddNewService, err)
	}

	return m.StartProcesses()
}

func (m *manager) MaxFileNameLen() int {
	return m.MaxFileNameLen()
}

func (m *manager) MaxFileTypeLen() int {
	return m.MaxFileTypeLen()
}

func (m *manager) MaxFileSize() int {
	return m.MaxFileSize()
}

func (m *manager) MaxPreviewSize() int {
	return m.MaxPreviewSize()
}

func (m *manager) Send(fileName, fileType string, fileData []byte,
	recipient *id.ID, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration,
	sendNew ft.SendNew) (*ftCrypto.TransferID, error) {
	return m.ft.Send(fileName, fileType, fileData, recipient, retry, preview,
		progressCB, period, sendNew)
}

func (m *manager) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {
	return m.ft.RegisterSentProgressCallback(tid, progressCB, period)
}

func (m *manager) CloseSend(tid *ftCrypto.TransferID) error {
	return m.CloseSend(tid)
}

func (m *manager) HandleIncomingTransfer(fileName string,
	key *ftCrypto.TransferKey, transferMAC []byte, numParts uint16, size uint32,
	retry float32, progressCB ft.ReceivedProgressCallback,
	period time.Duration) (*ftCrypto.TransferID, error) {
	return m.ft.HandleIncomingTransfer(
		fileName, key, transferMAC, numParts, size, retry, progressCB, period)
}

func (m *manager) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.ReceivedProgressCallback, period time.Duration) error {
	return m.ft.RegisterReceivedProgressCallback(tid, progressCB, period)
}

func (m *manager) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	return m.Receive(tid)
}
