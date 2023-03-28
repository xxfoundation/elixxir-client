////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"github.com/pkg/errors"
)

var (
	// InvalidReaction is returned if the passed reaction string is invalid.
	InvalidReaction = errors.New(
		"The reaction is not valid, it must be a single emoji")
)

// SupportedEmojis returns a list of emojis that are supported by the backend.
// The list includes all emojis described in [UTS #51 section A.1: Data Files].
//
// [UTS #51 section A.1: Data Files]: https://www.unicode.org/reports/tr51/#Data_Files
func SupportedEmojis() []Emoji {
	emojis := make([]Emoji, 0, len(emojiFile.Map))
	for _, emoji := range emojiFile.Map {
		emojis = append(emojis, emoji)
	}
	return emojis
}

// SupportedEmojisMap returns a map of emojis that are supported by the backend
// as described by [SupportedEmojis].
func SupportedEmojisMap() Map {
	// Make a copy of the map
	emojis := make(Map, len(emojiFile.Map))
	for c, emoji := range emojiFile.Map {
		emojis[c] = emoji
	}
	return emojis
}

// ValidateReaction checks that the reaction only contains a single emoji.
// Returns [InvalidReaction] if the emoji is invalid.
func ValidateReaction(reaction string) error {
	if _, exists := emojiFile.Map[reaction]; !exists {
		return InvalidReaction
	}

	return nil
}

// Map lists all emojis keyed on their character string.
type Map map[string]Emoji

// File represents the contents of an emoji file downloaded from Unicode.
type File struct {
	// Date is the date on the document
	Date string `json:"date"`

	// DateAccessed is the timestamp the file was downloaded
	DateAccessed string `json:"dateAccessed"`

	// Version is the version of Emoji described.
	Version string `json:"version"`

	// Map of all emoji character.
	Map Map `json:"map"`
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
