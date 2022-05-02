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
)

// manager handles the sending and receiving of file transfers for group chats.
type manager struct {
	// Callback that is called every time a new file transfer is received
	receiveCB ft.ReceiveCallback

	// File transfer manager
	ft ft.FileTransfer

	// Group chat manager
	gc groupChat.Manager

	myID *id.ID
	cmix ft.Cmix
}

// NewManager generates a new file transfer manager for group chat.
func NewManager(receiveCB ft.ReceiveCallback, params ft.Params, myID *id.ID,
	gc groupChat.Manager, cmix ft.Cmix, kv *versioned.KV,
	rng *fastRNG.StreamGenerator) (ft.FileTransfer, error) {

	sendNewCb := func(recipient *id.ID, info *ft.TransferInfo) error {
		return nil
	}

	sendEndCb := func(recipient *id.ID) {

	}

	ftManager, err := ft.NewManager(
		sendNewCb, sendEndCb, params, myID, cmix, kv, rng)
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

func (m manager) StartProcesses() (stoppable.Stoppable, error) {
	// TODO implement me
	panic("implement me")
}

func (m manager) MaxFileNameLen() int {
	return m.MaxFileNameLen()
}

func (m manager) MaxFileTypeLen() int {
	return m.MaxFileTypeLen()
}

func (m manager) MaxFileSize() int {
	return m.MaxFileSize()
}

func (m manager) MaxPreviewSize() int {
	return m.MaxPreviewSize()
}

func (m manager) Send(fileName, fileType string, fileData []byte,
	recipient *id.ID, retry float32, preview []byte,
	progressCB ft.SentProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {
	// TODO implement me
	panic("implement me")
}

func (m manager) RegisterSentProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.SentProgressCallback, period time.Duration) error {
	// TODO implement me
	panic("implement me")
}

func (m manager) CloseSend(tid *ftCrypto.TransferID) error {
	// TODO implement me
	panic("implement me")
}

func (m manager) AddNew(fileName string, key *ftCrypto.TransferKey,
	transferMAC []byte, numParts uint16, size uint32, retry float32,
	progressCB ft.ReceivedProgressCallback, period time.Duration) (
	*ftCrypto.TransferID, error) {
	// TODO implement me
	panic("implement me")
}

func (m manager) RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
	progressCB ft.ReceivedProgressCallback, period time.Duration) error {
	// TODO implement me
	panic("implement me")
}

func (m manager) Receive(tid *ftCrypto.TransferID) ([]byte, error) {
	// TODO implement me
	panic("implement me")
}
