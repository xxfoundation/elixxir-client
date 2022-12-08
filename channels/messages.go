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
	"gitlab.com/xx_network/primitives/id"
)

// userMessageInternal is the internal structure of a UserMessage protobuf.
type userMessageInternal struct {
	userMessage    *UserMessage
	channelMessage *ChannelMessage
	messageID      channel.MessageID
}

func newUserMessageInternal(
	ursMsg *UserMessage, chID *id.ID) (*userMessageInternal, error) {
	chanMessage := &ChannelMessage{}
	err := proto.Unmarshal(ursMsg.Message, chanMessage)
	if err != nil {
		return nil, err
	}

	channelMessage := chanMessage
	return &userMessageInternal{
		userMessage:    ursMsg,
		channelMessage: channelMessage,
		messageID:      channel.MakeMessageID(ursMsg.Message, chID),
	}, nil
}

func unmarshalUserMessageInternal(
	usrMsg []byte, channelID *id.ID) (*userMessageInternal, error) {

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
		messageID:      channel.MakeMessageID(um.Message, channelID),
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
func (umi *userMessageInternal) GetMessageID() channel.MessageID {
	return umi.messageID
}
