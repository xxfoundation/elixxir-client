package channels

import (
	"github.com/forPelevin/gomoji"
	"github.com/pkg/errors"
)

var InvalidReaction = errors.New("The reaction is not valid, " +
	"it must be a single emoji")

// ValidateReaction checks that the reaction only contains a single Emoji
func ValidateReaction(reaction string) error {
	if len(gomoji.RemoveEmojis(reaction)) > 0 {
		return InvalidReaction
	}

	if len(gomoji.FindAll(reaction)) != 1 {
		return InvalidReaction
	}

	return nil
}
