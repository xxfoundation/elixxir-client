////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import "github.com/golang/protobuf/proto"

// UserMessageInternal is the internal structure of a UserMessage protobuf.
type UserMessageInternal struct {
	UserMessage
	channelMsg ChannelMessage
}

// GetChannelMessage retrieves a serializes ChannelMessage within the
// UserMessageInternal object.
func (umi *UserMessageInternal) GetChannelMessage() ([]byte, error) {
	return proto.Marshal(&umi.channelMsg)
}
