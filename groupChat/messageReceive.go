////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"time"
)

// MessageReceive contains the GroupChat message and associated data that a user
// receives when getting a group message.
type MessageReceive struct {
	GroupID   *id.ID
	ID        group.MessageID
	Payload   []byte
	SenderID  *id.ID
	Timestamp time.Time
}

// String returns the MessageReceive as readable text. This functions adheres to
// the fmt.Stringer interface.
func (mr MessageReceive) String() string {
	groupID := "<nil>"
	if mr.GroupID != nil {
		groupID = mr.GroupID.String()
	}

	payload := "<nil>"
	if mr.Payload != nil {
		payload = fmt.Sprintf("%q", mr.Payload)
	}

	senderID := "<nil>"
	if mr.SenderID != nil {
		senderID = mr.SenderID.String()
	}

	str := []string{
		"GroupID:" + groupID,
		"ID:" + mr.ID.String(),
		"Payload:" + payload,
		"SenderID:" + senderID,
		"Timestamp:" + mr.Timestamp.String(),
	}

	return "{" + strings.Join(str, " ") + "}"
}
