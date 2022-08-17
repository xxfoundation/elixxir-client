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
	mux sync.Mutex
	*UserMessage
	channelMsg *ChannelMessage
}

func NewUserMessageInternal(ursMsg *UserMessage) *UserMessageInternal {
	return &UserMessageInternal{
		mux:         sync.Mutex{},
		UserMessage: ursMsg,
		channelMsg:  nil,
	}
}

// GetChannelMessage retrieves a serializes ChannelMessage within the
// UserMessageInternal object.
func (umi *UserMessageInternal) GetChannelMessage() (*ChannelMessage, error) {
	umi.mux.Lock()
	defer umi.mux.Unlock()

	// check if channel message
	if umi.channelMsg != nil {
		return umi.channelMsg, nil
	}
	// if not, deserialize and store
	unmarshalledChannelMsg := &ChannelMessage{}
	err := proto.Unmarshal(umi.UserMessage.Message, unmarshalledChannelMsg)
	if err != nil {
		return nil, err
	}

	return unmarshalledChannelMsg, nil
}
