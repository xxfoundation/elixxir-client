///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"strconv"
	"strings"
	"time"
)

// MessageReceive contains the GroupChat message and associated data that a user
// receives when getting a group message.
type MessageReceive struct {
	GroupID        *id.ID
	ID             group.MessageID
	Payload        []byte
	SenderID       *id.ID
	RecipientID    *id.ID
	EphemeralID    ephemeral.Id
	Timestamp      time.Time
	RoundID        id.Round
	RoundTimestamp time.Time
}

// String returns the MessageReceive as readable text. This functions satisfies
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

	recipientID := "<nil>"
	if mr.RecipientID != nil {
		recipientID = mr.RecipientID.String()
	}

	str := make([]string, 0, 9)
	str = append(str, "GroupID:"+groupID)
	str = append(str, "ID:"+mr.ID.String())
	str = append(str, "Payload:"+payload)
	str = append(str, "SenderID:"+senderID)
	str = append(str, "RecipientID:"+recipientID)
	str = append(str, "EphemeralID:"+strconv.FormatInt(mr.EphemeralID.Int64(), 10))
	str = append(str, "Timestamp:"+mr.Timestamp.String())
	str = append(str, "RoundID:"+strconv.FormatUint(uint64(mr.RoundID), 10))
	str = append(str, "RoundTimestamp:"+mr.RoundTimestamp.String())

	return "{" + strings.Join(str, " ") + "}"
}
