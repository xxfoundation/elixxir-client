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

	// WrongPrivateKeyErr is returned when the private key does not match the
	// channel's public key.
	WrongPrivateKeyErr = errors.New(
		"the passed private key does not match the channel")

	// WrongPasswordErr is returned when the encrypted packet could not be
	// decrypted using the supplied password.
	WrongPasswordErr = errors.New(
		"incorrect password")

	// MessageTypeAlreadyRegistered is returned if a handler has already been
	// registered with the supplied message type. Only one handler can be
	// registered per type.
	MessageTypeAlreadyRegistered = errors.New(
		"the given message type has already been registered")

	// InvalidReaction is returned if the passed reaction string is an invalid
	// emoji.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single emoji")
)
