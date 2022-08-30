////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/crypto/channel"
)

// userMessageInternal is the internal structure of a UserMessage protobuf.
type userMessageInternal struct {
	userMessage    *UserMessage
	channelMessage *ChannelMessage
	messageID      channel.MessageID
}

func newUserMessageInternal(ursMsg *UserMessage) (*userMessageInternal, error) {
	chanMessage := &ChannelMessage{}
	err := proto.Unmarshal(ursMsg.Message, chanMessage)
	if err != nil {
		return nil, err
	}

	channelMessage := chanMessage
	return &userMessageInternal{
		userMessage:    ursMsg,
		channelMessage: channelMessage,
		messageID:      channel.MakeMessageID(ursMsg.Message),
	}, nil
}

func unmarshalUserMessageInternal(usrMsg []byte) (*userMessageInternal, error) {

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

	return &userMessageInternal{
		userMessage:    um,
		channelMessage: channelMessage,
		messageID:      channel.MakeMessageID(um.Message),
	}, nil
}

// GetUserMessage retrieves the UserMessage within
// userMessageInternal.
func (umi *userMessageInternal) GetUserMessage() *UserMessage {
	return umi.userMessage
}

// GetChannelMessage retrieves the ChannelMessage within
// userMessageInternal.
func (umi *userMessageInternal) GetChannelMessage() *ChannelMessage {
	return umi.channelMessage
}

// GetMessageID retrieves the messageID for the message.
func (umi *userMessageInternal) GetMessageID() channel.MessageID {
	return umi.messageID
}
