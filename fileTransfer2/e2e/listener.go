////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/receive"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// Error messages.
const (
	// listener.Hear
	errProtoUnmarshal      = "[FT] Failed to proto unmarshal new file transfer request: %+v"
	errNewReceivedTransfer = "[FT] Failed to add new received transfer for %q: %+v"
)

// Name of listener (used for debugging)
const listenerName = "NewFileTransferListener-E2E"

// listener waits for a message indicating a new file transfer is starting.
// Adheres to the receive.Listener interface.
type listener struct {
	m *manager
}

// Hear is called when a new file transfer is received. It creates a new
// internal received file transfer and starts waiting to receive file part
// messages.
func (l *listener) Hear(msg receive.Message) {
	// Unmarshal the request message
	newFT := &NewFileTransfer{}
	err := proto.Unmarshal(msg.Payload, newFT)
	if err != nil {
		jww.ERROR.Printf(errProtoUnmarshal, err)
		return
	}

	transferKey := ftCrypto.UnmarshalTransferKey(newFT.GetTransferKey())

	// Add new transfer to start receiving parts
	tid, err := l.m.AddNew(newFT.FileName, &transferKey, newFT.TransferMac,
		uint16(newFT.NumParts), newFT.Size, newFT.Retry, nil, 0)
	if err != nil {
		jww.ERROR.Printf(errNewReceivedTransfer, newFT.FileName, err)
		return
	}

	// Call the reception callback
	go l.m.receiveCB(tid, newFT.FileName, newFT.FileType, msg.Sender,
		newFT.Size, newFT.Preview)
}

// Name returns a name used for debugging.
func (l *listener) Name() string {
	return listenerName
}
