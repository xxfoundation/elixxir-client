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
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// sendNewFileTransferMessage
	errMarshalInfo        = "failed to marshal new transfer info: %+v"
	errNewFtSendGroupChat = "failed to send initial file transfer message via group chat: %+v"
)

// sendNewFileTransferMessage sends a group chat message to the group ID
// informing them of the incoming file transfer.
func sendNewFileTransferMessage(
	groupID *id.ID, info *ft.TransferInfo, gc GroupChat) error {

	// Marshal the message
	payload, err := info.Marshal()
	if err != nil {
		return errors.Errorf(errMarshalInfo, err)
	}

	// Send the message via group chat
	_, _, _, err = gc.Send(groupID, newFileTransferTag, payload)
	if err != nil {
		return errors.Errorf(errNewFtSendGroupChat, err)
	}

	return nil
}
