////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import "testing"

/*
func TestValidateReaction(t *testing.T) {

	testReactions := []string{"🍆", "😂", "❤", "🤣", "👍", "😭", "🙏", "😘", "🥰",
		"😍", "😊", "☺", "A", "b", "AA", "1", "🍆🍆", "🍆A", "👍👍👍", "👍😘A",
		"O", "\u0000", "\u0011", "\u001F", "\u007F", "\u0080", "\u008A",
		"\u009F"}

	expected := []error{
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction, InvalidReaction, InvalidReaction, InvalidReaction}

	for i, r := range testReactions {
		err := ValidateReaction(r)
		if err != expected[i] {
			t.Errorf("Got incorrect response for `%s` (%d): "+
				"`%s` vs `%s`", r, i, err, expected[i])
		}
	}
}*/

func TestValidateReaction(t *testing.T) {
	testReactions := []string{
		"🍆", "😂", "❤", "🤣", "👍", "😭", "🙏", "😘", "🥰", "😍", "😊",
		"☺", "A", "b", "AA", "1", "🍆🍆", "🍆A", "👍👍👍", "👍😘A",
	}

	expected := []error{
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		InvalidReaction, nil, InvalidReaction, InvalidReaction, InvalidReaction,
		InvalidReaction,
	}

	for i, r := range testReactions {
		err := ValidateReaction(r)
		if err != expected[i] {
			t.Errorf("Got incorrect response for %q (%d): "+
				"`%s` vs `%s`", r, i, err, expected[i])
		}
	}
}
