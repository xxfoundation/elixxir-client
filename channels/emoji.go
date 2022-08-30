////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bufio"
	"bytes"
	jww "github.com/spf13/jwalterweatherman"
	"regexp"
)

//based on emojis found at https://unicode.org/emoji/charts/full-emoji-list.html
const findEmoji = `[\xA9\xAE\x{2000}-\x{3300}\x{1F000}-\x{1FBFF}]`

var compiledRegex *regexp.Regexp

// compile the regular expression in an init so it is only
// compiled once
func init() {
	var err error
	compiledRegex, err = regexp.Compile(findEmoji)
	if err != nil {
		jww.FATAL.Panicf("failed to compile the regex for emoji finding " +
			"within channels")
	}
}

// ValidateReaction checks that the reaction only contains a single emoji.
func ValidateReaction(reaction string) error {

	//make sure it is only only character
	reactRunes := []rune(reaction)
	if len(reactRunes) > 1 {
		return InvalidReaction
	}

	reader := bufio.NewReader(bytes.NewReader([]byte(reaction)))

	// make sure it has emojis
	if !compiledRegex.MatchReader(reader) {
		return InvalidReaction
	}

	return nil
}
