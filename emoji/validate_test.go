////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"reflect"
	"testing"
)

// Unit test of SupportedEmojis.
func TestSupportedEmojis(t *testing.T) {
	emojis := SupportedEmojis()

	if len(emojis) != len(emojiMap) {
		t.Errorf("Incorrect number of emojis.\nexpected: %d\nreceived: %d",
			len(emojiMap), len(emojis))
	}
}

// Unit test of SupportedEmojisMap.
func TestSupportedEmojisMap(t *testing.T) {
	emojis := SupportedEmojisMap()

	if !reflect.DeepEqual(emojis, emojiMap) {
		t.Errorf("Incorrect map.\nexpected: %v\nreceived: %v",
			emojiMap, emojis)
	}
}

var tests = []struct {
	Name  string
	Input []string
	Errs  map[string]error
}{
	{
		Name: "Single-rune emojis",
		Input: []string{"😀", "👋", "🍆", "😂", "❤", "🤣", "👍", "😭", "🙏",
			"😘", "🥰", "😍", "😊", "☺", "🏴"},
	}, {
		Name:  "Multi-rune emojis",
		Input: []string{"👱‍♂️", "👋🏿", "🧏‍♀️", "❤️", "👩🏽‍❤️‍💋‍👨🏽", "🏴‍☠️"},
	}, {
		Name:  "Long multi-rune emojis",
		Input: []string{"👨🏻‍👩🏻‍👦🏻‍👦🏻"},
		Errs:  map[string]error{"Test_validateEmoji": InvalidReaction}, // Note: This shouldn't error. The emoji list does not support arbitrary ordering of modifiers
	}, {
		Name:  "Multiple single-rune emojis",
		Input: []string{"😀👋", "😀😀", "🍆🍆", "👍👍👍"},
		Errs: map[string]error{
			"TestValidateReaction": InvalidReaction,
			"Test_validateEmoji":   InvalidReaction},
	}, {
		Name:  "Multiple character strings",
		Input: []string{"🧖 hello 🦋 world", "😀 hello 😀 world"},
		Errs: map[string]error{
			"TestValidateReaction": InvalidReaction,
			"Test_validateEmoji":   InvalidReaction},
	}, {
		Name:  "Single normal characters",
		Input: []string{"A", "b", "1"},
		Errs:  map[string]error{"Test_validateEmoji": InvalidReaction},
	}, {
		Name:  "Multiple normal characters",
		Input: []string{"AA", "badaw"},
		Errs: map[string]error{
			"TestValidateReaction": InvalidReaction,
			"Test_validateEmoji":   InvalidReaction},
	}, {
		Name:  "Multiple normal characters and emojis",
		Input: []string{"🍆A", "👍😘A"},
		Errs: map[string]error{
			"TestValidateReaction": InvalidReaction,
			"Test_validateEmoji":   InvalidReaction},
	}, {
		Name:  "No characters",
		Input: []string{""},
		Errs: map[string]error{
			"TestValidateReaction": InvalidReaction,
			"Test_validateEmoji":   InvalidReaction},
	},
}

// Unit test of ValidateReaction.
func TestValidateReaction(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			for i, r := range tt.Input {
				err := ValidateReaction(r)
				if err != tt.Errs["TestValidateReaction"] {
					t.Errorf("%2d. Incorrect response for reaction %q %X."+
						"\nexpected: %s\nreceived: %s",
						i, r, []rune(r), tt.Errs["TestValidateReaction"], err)
				}
			}
		})
	}
}

// Unit test of validateEmoji.
func Test_validateEmoji(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			for i, r := range tt.Input {
				err := validateEmoji(r)
				if err != tt.Errs["Test_validateEmoji"] {
					t.Errorf("%2d. Incorrect response for reaction %q %X."+
						"\nexpected: %s\nreceived: %s",
						i, r, []rune(r), tt.Errs["Test_validateEmoji"], err)
				}
			}
		})
	}
}
