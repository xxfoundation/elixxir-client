////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
)

// Error messages.
const (
	// sendNewFileTransferMessage
	errNewFtSendGroupChat = "failed to send initial file transfer message via group chat: %+v"
)

// sendNewFileTransferMessage sends a group chat message to the group ID
// informing them of the incoming file transfer.
func sendNewFileTransferMessage(
	groupID *id.ID, transferInfo []byte, gc gcManager) error {

	// Send the message via group chat
	_, _, _, err := gc.Send(groupID, newFileTransferTag, transferInfo)
	if err != nil {
		return errors.Errorf(errNewFtSendGroupChat, err)
	}

	return nil
}
