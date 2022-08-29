package channels

import "github.com/pkg/errors"

var (
	ChannelAlreadyExistsErr = errors.New(
		"the channel cannot be added because it already exists")
	ChannelDoesNotExistsErr = errors.New(
		"the channel cannot be found")
	MessageTooLongErr = errors.New(
		"the passed message is too long")
)
