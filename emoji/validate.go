// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

/*// Based on emojis found at
// https://unicode.org/emoji/charts/full-emoji-list.html
const findEmoji = `[\xA9\xAE\x{2000}-\x{3300}\x{1F000}-\x{1FBFF}]`

// compiledFindEmoji is a regular expression for matching an emoji.
var compiledFindEmoji = regexp.MustCompile(findEmoji)*/

// ValidateReaction checks that the reaction only contains a single emoji.
func ValidateReaction(reaction string) error {

	// Make sure it is the only character
	reactRunes := []rune(reaction)
	if len(reactRunes) > 1 {
		return InvalidReaction
	}

	/*
		reader := bytes.NewReader([]byte(reaction))
		// Make sure it has emojis
		if !compiledFindEmoji.MatchReader(reader) {
			return InvalidReaction
		}
	*/
	return nil
}
