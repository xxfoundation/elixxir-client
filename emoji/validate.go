////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"fmt"
	"github.com/forPelevin/gomoji"
	"github.com/pkg/errors"
)

var (
	// InvalidReaction is returned if the passed reaction string is an invalid
	// emoji.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single emoji")
)

// fixme: for the above error print, maybe simply say it is an unsupported emoji?

// fixme: alternative solution for emoji problem: expose this to frontned and have them call it

// ValidateReaction checks that the reaction only contains a single emoji.
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
		fmt.Print(emojisList[0].Character)
		return InvalidReaction
	}

	return nil
}
