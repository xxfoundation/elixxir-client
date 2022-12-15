////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import "github.com/pkg/errors"

var (
	// ChannelAlreadyExistsErr is returned when attempting to join a channel
	// that the user is already in.
	ChannelAlreadyExistsErr = errors.New(
		"the channel cannot be added because it already exists")

	// ChannelDoesNotExistsErr is returned when a channel does not exist.
	ChannelDoesNotExistsErr = errors.New("the channel cannot be found")

	// MessageTooLongErr is returned when attempting to send a message that is
	// too large.
	MessageTooLongErr = errors.New("the passed message is too long")

	// WrongPrivateKey is returned when the private key does not match the
	// channel's public key.
	WrongPrivateKey = errors.New(
		"the passed private key does not match the channel")

	// MessageTypeAlreadyRegistered is returned if a handler has already been
	// registered with the supplied message type. Only one handler can be
	// registered per type.
	MessageTypeAlreadyRegistered = errors.New(
		"the given message type has already been registered")
)
