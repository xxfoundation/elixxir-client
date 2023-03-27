////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"github.com/pkg/errors"
	"github.com/rivo/uniseg"
)

var (
	// InvalidReaction is returned if the passed reaction string is an invalid
	// emoji.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single emoji")
)

// SupportedEmojis returns a list of emojis that are supported by the backend.
func SupportedEmojis() []Emoji {
	emojis := make([]Emoji, 0, len(emojiMap))
	for _, emoji := range emojiMap {
		emojis = append(emojis, emoji)
	}
	return emojis
}

// SupportedEmojisMap returns a map of emojis that are supported by the backend.
func SupportedEmojisMap() Map {
	emojis := make(Map, len(emojiMap))
	for c, emoji := range emojiMap {
		emojis[c] = emoji
	}
	return emojis
}

// ValidateReaction checks that the reaction only contains a single emoji.
// Returns InvalidReaction if the emoji is invalid.
func ValidateReaction(reaction string) error {
	if len(reaction) < 1 {
		// No characters found
		return InvalidReaction
	} else if uniseg.GraphemeClusterCount(reaction) > 1 {
		// More than one character found
		return InvalidReaction
	} else if _, exists := emojiMap[reaction]; !exists {
		// Character is not an emoji
		return InvalidReaction
	}

	return nil
}

// Emoji represents comprehensive information of each Unicode emoji character.
type Emoji struct {
	Character string `json:"character"` // Actual Unicode character
	Name      string `json:"name"`      // CLDR short name
	Comment   string `json:"comment"`   // Data file comments; usually version
	CodePoint string `json:"codePoint"` // Code point(s) for character
	Group     string `json:"group"`
	Subgroup  string `json:"subgroup"`
}

// Map lists all emojis keyed on their character string.
type Map map[string]Emoji

