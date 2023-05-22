////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

// userMessageInternal is the internal structure of a UserMessage protobuf.
type userMessageInternal struct {
	userMessage    *UserMessage
	channelMessage *ChannelMessage
	messageID      message.ID
	messageType    MessageType
}

func newUserMessageInternal(
	ursMsg *UserMessage, chID *id.ID, messageType MessageType) (*userMessageInternal, error) {
	chanMessage := &ChannelMessage{}
	err := proto.Unmarshal(ursMsg.Message, chanMessage)
	if err != nil {
		return nil, err
	}

	channelMessage := chanMessage
	return &userMessageInternal{
		userMessage:    ursMsg,
		channelMessage: channelMessage,
		messageID: message.DeriveChannelMessageID(chID,
			chanMessage.RoundID,
			ursMsg.Message),
		messageType: messageType,
	}, nil
}

func unmarshalUserMessageInternal(usrMsg []byte, channelID *id.ID,
	messageType MessageType) (*userMessageInternal, error) {

	um := &UserMessage{}
	if err := proto.Unmarshal(usrMsg, um); err != nil {
		return nil, err
	}

	channelMessage := &ChannelMessage{}
	err := proto.Unmarshal(um.Message, channelMessage)
	if err != nil {
		return nil, err
	}

	return &userMessageInternal{
		userMessage:    um,
		channelMessage: channelMessage,
		messageID: message.DeriveChannelMessageID(channelID,
			channelMessage.RoundID, um.Message),
		messageType: messageType,
	}, nil
}

// GetUserMessage retrieves the UserMessage within userMessageInternal.
func (umi *userMessageInternal) GetUserMessage() *UserMessage {
	return umi.userMessage
}

// GetChannelMessage retrieves the ChannelMessage within userMessageInternal.
func (umi *userMessageInternal) GetChannelMessage() *ChannelMessage {
	return umi.channelMessage
}

// GetMessageID retrieves the messageID for the message.
func (umi *userMessageInternal) GetMessageID() message.ID {
	return umi.messageID
}

// GetMessageType retrieves the messageType for the message.
func (umi *userMessageInternal) GetMessageType() MessageType {
	return umi.messageType
}

// String adheres to the fmt.Stringer interface.
func (umi *userMessageInternal) String() string {
	fields := []string{
		"userMessage:{" + umi.userMessage.String() + "}",
		"channelMessage:{" + umi.channelMessage.String() + "}",
		"messageID:" + umi.messageID.String(),
		"messageType:" + umi.messageType.String(),
	}

	return "{" + strings.Join(fields, " ") + "}"
}
