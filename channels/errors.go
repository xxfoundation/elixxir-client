////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import "github.com/pkg/errors"

var (
	ChannelAlreadyExistsErr = errors.New(
		"the channel cannot be added because it already exists")
	ChannelDoesNotExistsErr = errors.New(
		"the channel cannot be found")
	MessageTooLongErr = errors.New(
		"the passed message is too long")
	WrongPrivateKey = errors.New(
		"the passed private key does not match the channel")
	MessageTypeAlreadyRegistered = errors.New("the given message type has " +
		"already been registered")
)
