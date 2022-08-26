package channels

import "github.com/pkg/errors"

var ChannelAlreadyExistsErr = errors.New("the channel cannot be added " +
	"becasue it already exists")

var ChannelDoesNotExistsErr = errors.New("the channel cannot be found")

var MessageTooLongErr = errors.New("the passed message is too long")
