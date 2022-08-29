package channels

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

var InvalidReaction = errors.New("The reaction is not valid, " +
	"it must be a single emoji")

// ValidateReaction checks that the reaction only contains a single Emoji
func ValidateReaction(reaction string) error {
	jww.WARN.Printf("Reaction Validation Not Yet Implemented")
	return nil
}
