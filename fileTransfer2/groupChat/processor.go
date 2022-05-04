////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/elixxir/client/groupChat"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	// processor.Process
	errProtoUnmarshal      = "[FT] Failed to proto unmarshal new file transfer request: %+v"
	errNewReceivedTransfer = "[FT] Failed to add new received transfer for %q: %+v"
)

type processor struct {
	*Manager
}

func (p *processor) Process(decryptedMsg groupChat.MessageReceive,
	_ format.Message, _ receptionID.EphemeralIdentity, _ rounds.Round) {
	// Unmarshal the request message
	info, err := ft.UnmarshalTransferInfo(decryptedMsg.Payload)
	if err != nil {
		jww.ERROR.Printf(errProtoUnmarshal, err)
		return
	}

	// Add new transfer to start receiving parts
	tid, err := p.HandleIncomingTransfer(info.FileName, &info.Key, info.Mac,
		info.NumParts, info.Size, info.Retry, nil, 0)
	if err != nil {
		jww.ERROR.Printf(errNewReceivedTransfer, info.FileName, err)
		return
	}

	// Call the reception callback
	go p.receiveCB(tid, info.FileName, info.FileType, decryptedMsg.SenderID,
		info.Size, info.Preview)
}

func (p *processor) String() string {
	return "GroupFileTransfer"
}
