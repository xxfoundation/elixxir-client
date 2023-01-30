////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"github.com/forPelevin/gomoji"
	"github.com/pkg/errors"
)

var (
	// InvalidReaction is returned if the passed reaction string is an invalid
	// emoji.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single emoji")
)

// SupportedEmojis returns a list of emojis that are supported by the backend.
func SupportedEmojis() []gomoji.Emoji {
	return gomoji.AllEmojis()
}

// ValidateReaction checks that the reaction only contains a single emoji.
// Returns InvalidReaction if the emoji is invalid.
func ValidateReaction(reaction string) error {
	emojisList := gomoji.CollectAll(reaction)
	if len(emojisList) < 1 {
		// No emojis found
		return InvalidReaction
	} else if len(emojisList) > 1 {
		// More than one emoji found
		return InvalidReaction
	} else if emojisList[0].Character != reaction {
		// Non-emoji characters found alongside an emoji
		return InvalidReaction
	}

	return nil
}
