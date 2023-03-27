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
//	    "slug": "smiling-face",
//	    "character": "☺️",
//	    "unicode_name": "E0.6 smiling face",
//	    "code_point": "263A FE0F",
//	    "group": "Smileys \u0026 Emotion",
//	    "sub_group": "face-affection"
//	  },
//	  {
//	    "slug": "frowning-face",
//	    "character": "☹️",
//	    "unicode_name": "E0.7 frowning face",
//	    "code_point": "2639 FE0F",
//	    "group": "Smileys \u0026 Emotion",
//	    "sub_group": "face-concerned"
//	  },
//	  {
//	    "slug": "banana",
//	    "character": "�",
//	    "unicode_name": "E0.6 banana",
//	    "code_point": "1F34C",
//	    "group": "Food \u0026 Drink",
//	    "sub_group": "food-fruit"
//	  }
//	]
func SupportedEmojis() ([]byte, error) {
	return json.Marshal(emoji.SupportedEmojis())
}

// SupportedEmojisMap returns a map of emojis that are supported by the backend.
//
// Returns:
//   - []byte - JSON of a map of emoji.Emoji.
//
// Example JSON:
//
//  {
//    "☹️": {
//      "character": "☹️",
//      "name": "frowning face",
//      "comment": "E0.7",
//      "codePoint": "2639 FE0F",
//      "group": "Smileys \u0026 Emotion",
//      "subgroup": "face-concerned"
//    },
//    "☺️": {
//      "character": "☺️",
//      "name": "smiling face",
//      "comment": "E0.6",
//      "codePoint": "263A FE0F",
//      "group": "Smileys \u0026 Emotion",
//      "subgroup": "face-affection"
//    },
//    "☢️": {
//      "character": "☢️",
//      "name": "radioactive",
//      "comment": "E1.0",
//      "codePoint": "2622 FE0F",
//      "group": "Symbols",
//      "subgroup": "warning"
//    },
//  }
func SupportedEmojisMap() ([]byte, error) {
	return json.Marshal(emoji.SupportedEmojisMap())
}

// ValidateReaction checks that the reaction only contains a single emoji.
//
// Parameters:
//   - reaction - The reaction emoji to validate.
//
// Returns:
//   - Error emoji.InvalidReaction if the reaction is not valid and nil
//     otherwise.
func ValidateReaction(reaction string) error {
	return emoji.ValidateReaction(reaction)
}
