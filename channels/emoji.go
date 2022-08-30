package channels

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"regexp"
)

// found at https://www.regextester.com/106421
const findEmoji = `(\\u00a9|\\u00ae|[\\u2000-\\u3300]|\\ud83c[\\ud000-` +
	`\\udfff]|\\ud83d[\\ud000-\\udfff]|\\ud83e[\\ud000-\\udfff])`

var InvalidReaction = errors.New(
	"The reaction is not valid, it must be a single emoji")

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

	// make sure it is only one emoji
	if !compiledRegex.Match([]byte(reaction)) {
		return InvalidReaction
	}

	return nil
}
