package channels

import (
	"github.com/pkg/errors"
	"regexp"
)

const findEmoji = "(\\\\u00a9|\\\\u00ae|[\\\\u2000-\\\\u3300]|\\\\ud83c[\\\\ud000-\\\\udfff]|\\\\ud83d[\\\\ud000-\\\\udfff]|\\\\ud83e[\\\\ud000-\\\\udfff])"

var InvalidReaction = errors.New(
	"The reaction is not valid, it must be a single emoji")

// ValidateReaction checks that the reaction only contains a single emoji.
func ValidateReaction(reaction string) error {
	reactRunes := []rune(reaction)
	if len(reactRunes) > 1 {
		return InvalidReaction
	}

	reg, err := regexp.Compile(findEmoji)
	if err != nil {
		return err
	}

	if !reg.Match([]byte(reaction)) {
		return InvalidReaction
	}

	/*
		fmt.Println(string(reactRunes[0]))
		fmt.Println(reactRunes[1])
		fmt.Println(reactRunes[2])

		jww.WARN.Printf("Reaction Validation Not Yet Implemented")*/
	return nil
}
