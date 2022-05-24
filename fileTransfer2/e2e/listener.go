////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/receive"
)

// Error messages.
const (
	// listener.Hear
	errNewReceivedTransfer = "[FT] Failed to add new received transfer: %+v"
)

// Name of listener (used for debugging)
const listenerName = "NewFileTransferListener-E2E"

// listener waits for a message indicating a new file transfer is starting.
// Adheres to the receive.Listener interface.
type listener struct {
	m *Wrapper
}

// Hear is called when a new file transfer is received. It creates a new
// internal received file transfer and starts waiting to receive file part
// messages.
func (l *listener) Hear(msg receive.Message) {
	// Add new transfer to start receiving parts
	tid, info, err := l.m.ft.HandleIncomingTransfer(msg.Payload, nil, 0)
	if err != nil {
		jww.ERROR.Printf(errNewReceivedTransfer, err)
		return
	}

	// Call the reception callback
	go l.m.receiveCB(
		tid, info.FileName, info.FileType, msg.Sender, info.Size, info.Preview)
}

// Name returns a name used for debugging.
func (l *listener) Name() string {
	return listenerName
}
