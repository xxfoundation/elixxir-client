////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"github.com/golang/protobuf/proto"
	"sync"
)

// UserMessageInternal is the internal structure of a UserMessage protobuf.
type UserMessageInternal struct {
	mux            sync.RWMutex
	userMessage    *UserMessage
	channelMessage *ChannelMessage
}

func NewUserMessageInternal(ursMsg *UserMessage) *UserMessageInternal {
	return &UserMessageInternal{
		mux:            sync.RWMutex{},
		userMessage:    ursMsg,
		channelMessage: nil,
	}
}

// GetUserMessage retrieves the UserMessage within
// UserMessageInternal.
func (umi *UserMessageInternal) GetUserMessage() *UserMessage {
	umi.mux.RLock()
	umi.mux.RUnlock()
	return umi.userMessage
}

// GetChannelMessage retrieves the ChannelMessage within
// UserMessageInternal. This is a lazy getter which will
// deserialize the ChannelMessage within the UserMessage.Message field.
// This deserialized ChannelMessage will then be placed into
// UserMessageInternal's channelMessage field and return. On subsequent calls it will return
// the message stored in UserMessageInternal.
func (umi *UserMessageInternal) GetChannelMessage() (*ChannelMessage, error) {
	umi.mux.Lock()
	defer umi.mux.Unlock()

	if umi.channelMessage == nil {
		chanMessage := &ChannelMessage{}
		err := proto.Unmarshal(umi.userMessage.Message, chanMessage)
		if err != nil {
			return nil, err
		}

		umi.channelMessage = chanMessage
	}

	return umi.channelMessage, nil
}
