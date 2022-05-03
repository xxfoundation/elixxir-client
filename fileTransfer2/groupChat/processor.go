////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/elixxir/client/groupChat"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/primitives/format"
)

// Error messages.
const (
	// processor.Process
	errProtoUnmarshal      = "[FT] Failed to proto unmarshal new file transfer request: %+v"
	errNewReceivedTransfer = "[FT] Failed to add new received transfer for %q: %+v"
)

type processor struct {
	*manager
}

func (p *processor) Process(decryptedMsg groupChat.MessageReceive,
	_ format.Message, _ receptionID.EphemeralIdentity, _ rounds.Round) {
	// Unmarshal the request message
	var newFT ft.NewFileTransfer
	err := proto.Unmarshal(decryptedMsg.Payload, &newFT)
	if err != nil {
		jww.ERROR.Printf(errProtoUnmarshal, err)
		return
	}

	transferKey := ftCrypto.UnmarshalTransferKey(newFT.GetTransferKey())

	// Add new transfer to start receiving parts
	tid, err := p.HandleIncomingTransfer(newFT.FileName, &transferKey,
		newFT.TransferMac, uint16(newFT.NumParts), newFT.Size, newFT.Retry,
		nil, 0)
	if err != nil {
		jww.ERROR.Printf(errNewReceivedTransfer, newFT.FileName, err)
		return
	}

	// Call the reception callback
	go p.receiveCB(tid, newFT.FileName, newFT.FileType, decryptedMsg.SenderID,
		newFT.Size, newFT.Preview)
}

func (p *processor) String() string {
	return "GroupFileTransfer"
}
