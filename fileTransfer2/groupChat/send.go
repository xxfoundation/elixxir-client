////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	ft "gitlab.com/elixxir/client/fileTransfer2"
	"gitlab.com/elixxir/client/groupChat"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// sendNewFileTransferMessage
	errProtoMarshal       = "failed to proto marshal NewFileTransfer: %+v"
	errNewFtSendGroupChat = "failed to send initial file transfer message via group chat: %+v"

	// sendEndFileTransferMessage
	errEndFtSendGroupChat = "[FT] Failed to send ending file transfer message via group chat: %+v"
)

// sendNewFileTransferMessage sends a group chat message to the group ID
// informing them of the incoming file transfer.
func sendNewFileTransferMessage(
	groupID *id.ID, info *ft.TransferInfo, gc groupChat.GroupChat) error {

	// Construct NewFileTransfer message
	protoMsg := &ft.NewFileTransfer{
		FileName:    info.FileName,
		FileType:    info.FileType,
		TransferKey: info.Key.Bytes(),
		TransferMac: info.Mac,
		NumParts:    uint32(info.NumParts),
		Size:        info.Size,
		Retry:       info.Retry,
		Preview:     info.Preview,
	}

	// Marshal the message
	payload, err := proto.Marshal(protoMsg)
	if err != nil {
		return errors.Errorf(errProtoMarshal, err)
	}

	// Send the message via group chat
	_, _, _, err = gc.Send(groupID, newFileTransferTag, payload)
	if err != nil {
		return errors.Errorf(errNewFtSendGroupChat, err)
	}

	return nil
}

// sendEndFileTransferMessage sends a group chat message to the group ID
// informing them that all file parts have arrived once the network is healthy.
func sendEndFileTransferMessage(groupID *id.ID, gc groupChat.GroupChat) {
	_, _, _, err := gc.Send(groupID, endFileTransferTag, nil)
	if err != nil {
		jww.ERROR.Printf(errEndFtSendGroupChat, err)
	}
}
