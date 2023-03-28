////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"

	"gitlab.com/elixxir/client/v4/emoji"
)

// SupportedEmojis returns a list of emojis that are supported by the backend.
//
// Returns:
//   - []byte - JSON of an array of emoji.Emoji.
//
// Example JSON:
//
//	[
//	  {
//      "character": "☹️",
//      "name": "frowning face",
//      "comment": "E0.7",
//      "codePoint": "2639 FE0F",
//      "group": "Smileys \u0026 Emotion",
//      "subgroup": "face-concerned"
//	  },
//	  {
//      "character": "☺️",
//      "name": "smiling face",
//      "comment": "E0.6",
//      "codePoint": "263A FE0F",
//      "group": "Smileys \u0026 Emotion",
//      "subgroup": "face-affection"
//	  },
//	  {
//      "character": "☢️",
//      "name": "radioactive",
//      "comment": "E1.0",
//      "codePoint": "2622 FE0F",
//      "group": "Symbols",
//      "subgroup": "warning"
//	  }
//	]
//
// Deprecated: ValidateReaction no longer checks this list to validate emojis.
func SupportedEmojis() ([]byte, error) {
	return json.Marshal(emoji.SupportedEmojis())
}

// ValidateReaction checks that the reaction only contains a single grapheme
// (one or more codepoints that appear as a single character to the user).
//
// Parameters:
//   - reaction - The reaction to validate.
//
// Returns:
//   - Error emoji.InvalidReaction if the reaction is not a single character.
func ValidateReaction(reaction string) error {
	return emoji.ValidateReaction(reaction)
}
