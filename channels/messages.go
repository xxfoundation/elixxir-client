////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/crypto/channel"
)

// UserMessageInternal is the internal structure of a UserMessage protobuf.
type UserMessageInternal struct {
	userMessage    *UserMessage
	channelMessage *ChannelMessage
	messageID      channel.MessageID
}

func NewUserMessageInternal(ursMsg *UserMessage) (*UserMessageInternal, error) {
	chanMessage := &ChannelMessage{}
	err := proto.Unmarshal(ursMsg.Message, chanMessage)
	if err != nil {
		return nil, err
	}

	channelMessage := chanMessage
	return &UserMessageInternal{
		userMessage:    ursMsg,
		channelMessage: channelMessage,
		messageID:      channel.MakeMessageID(ursMsg.Message),
	}, nil
}

func UnmarshalUserMessageInternal(usrMsg []byte) (*UserMessageInternal, error) {

	um := &UserMessage{}
	if err := proto.Unmarshal(usrMsg, um); err != nil {
		return nil, err
	}

	chanMessage := &ChannelMessage{}
	err := proto.Unmarshal(um.Message, chanMessage)
	if err != nil {
		return nil, err
	}

	channelMessage := chanMessage

	return &UserMessageInternal{
		userMessage:    um,
		channelMessage: channelMessage,
	}, nil
}

// GetUserMessage retrieves the UserMessage within
// UserMessageInternal.
func (umi *UserMessageInternal) GetUserMessage() *UserMessage {
	return umi.userMessage
}

// GetChannelMessage retrieves the ChannelMessage within
// UserMessageInternal.
func (umi *UserMessageInternal) GetChannelMessage() *ChannelMessage {
	return umi.channelMessage
}

// GetMessageID retrieves the messageID for the message.
func (umi *UserMessageInternal) GetMessageID() channel.MessageID {
	return umi.messageID
}
