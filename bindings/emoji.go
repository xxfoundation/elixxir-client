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
//   - []byte - JSON of an array of gomoji.Emoji.
func SupportedEmojis() ([]byte, error) {
	return json.Marshal(emoji.SupportedEmojis())
}

// ValidateReaction checks that the reaction only contains a single emoji.
//
// Parameters:
//   - reaction - The reaction to emoji to validate.
//
// Returns:
//   - Error emoji.InvalidReaction if the reaction is not valid and nil
//     otherwise.
func ValidateReaction(reaction string) error {
	return emoji.ValidateReaction(reaction)
}
