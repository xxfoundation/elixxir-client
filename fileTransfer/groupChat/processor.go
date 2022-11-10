////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/client/v5/groupChat"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	// processor.Process
	errNewReceivedTransfer = "[FT] Failed to add new received transfer: %+v"
)

// processor processes the incoming E2E new file transfer messages to start
// receiving a new file transfer. Adheres to the Processor interface.
type processor struct {
	*Wrapper
}

// Process receives new file transfer messages and registers it with the file
// transfer manager. Then the caller is notified of the file transfer via the
// reception callback. It is the responsibility of the caller to register a
// progress callback.
func (p *processor) Process(decryptedMsg groupChat.MessageReceive,
	_ format.Message, _ receptionID.EphemeralIdentity, _ rounds.Round) {
	// Add new transfer to start receiving parts
	tid, info, err := p.ft.HandleIncomingTransfer(decryptedMsg.Payload, nil, 0)
	if err != nil {
		jww.ERROR.Printf(errNewReceivedTransfer, err)
		return
	}

	// Call the reception callback
	go p.receiveCB(tid, info.FileName, info.FileType, decryptedMsg.SenderID,
		info.Size, info.Preview)
}

// String returns a human-readable identifier for this processor. Adheres to
// the fmt.Stringer interface.
func (p *processor) String() string {
	return "GroupFileTransfer"
}
