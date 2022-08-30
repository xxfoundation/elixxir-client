package channels

import (
	"testing"
)

func TestValidateReaction(t *testing.T) {

	testReactions := []string{"🍆", "😂", "❤", "🤣", "👍", "😭", "🙏", "😘", "🥰", "😍",
		"😊", "☺", "A", "b", "AA", "1", "🍆🍆", "🍆A", "👍👍👍", "👍😘A", "O"}
	expected := []error{
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction}

	for i, r := range testReactions {
		err := ValidateReaction(r)
		if err != expected[i] {
			t.Errorf("Got incorrect response for `%s` (%d): "+
				"`%s` vs `%s`", r, i, err, expected[i])
		}
	}
}
