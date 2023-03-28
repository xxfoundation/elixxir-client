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
	// InvalidReaction is returned if the passed reaction string is invalid.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single character")
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

// ValidateReaction checks that the reaction only contains a single grapheme
// (one or more codepoints that appear as a single character to the user).
// Returns InvalidReaction if the reaction is invalid.
func ValidateReaction(reaction string) error {
	if !isSingleGrapheme(reaction) {
		return InvalidReaction
	}

	return nil
}

// validateEmoji checks that the reaction only contains a single emoji.
// Returns InvalidReaction if the emoji is invalid.
func validateEmoji(emoji string) error {
	if !isSingleGrapheme(emoji) {
		// Incorrect number of graphemes
		return InvalidReaction
	} else if _, exists := emojiMap[emoji]; !exists {
		// Character is not an emoji
		return InvalidReaction
	}

	return nil
}

// isSingleGrapheme returns true if the string is a single grapheme or false if
// it is zero or more than one.
func isSingleGrapheme(s string) bool {
	if s == "" {
		return false
	}

	for n, state := 0, -1; len(s) > 0; n++ {
		_, s, _, state = uniseg.FirstGraphemeClusterInString(s, state)
		if n > 0 {
			return false
		}
	}
	return true
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
