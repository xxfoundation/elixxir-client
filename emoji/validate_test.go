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

	if len(emojis) != len(emojiFile.Map) {
		t.Errorf("Incorrect number of emojis.\nexpected: %d\nreceived: %d",
			len(emojiFile.Map), len(emojis))
	}
}

// Unit test of SupportedEmojisMap.
func TestSupportedEmojisMap(t *testing.T) {
	emojis := SupportedEmojisMap()

	if !reflect.DeepEqual(emojis, emojiFile.Map) {
		t.Errorf("Incorrect map.\nexpected: %v\nreceived: %v",
			emojiFile.Map, emojis)
	}
}

var tests = []struct {
	Name   string
	Input  []string
	Output error
}{
	{
		Name: "Single-rune emojis",
		Input: []string{"😀", "👋", "🍆", "😂", "❤", "🤣", "👍", "😭", "🙏",
			"😘", "🥰", "😍", "😊", "☺", "🏴"},
	}, {
		Name:  "Multi-rune emojis",
		Input: []string{"👋🏿", "❤️"},
	}, {
		Name:  "ZWJ Sequences",
		Input: []string{"👱‍♂️", "🧏‍♀️", "👩🏽‍❤️‍💋‍👨🏽", "🏴‍☠️"},
	}, {
		Name:   "Non-RGI ZWJ Sequences",
		Input:  []string{"👨🏻‍👩🏻‍👦🏻‍👦🏻", "⛑🏻", "👪🏿", "🤼🏻", "🏴󠁵󠁳󠁴󠁸󠁿", "👩🏽‍❤️‍🧑"},
		Output: InvalidReaction,
	}, {
		Name:   "Multiple Single-Rune Emojis",
		Input:  []string{"😀👋", "😀😀", "🍆🍆", "👍👍👍"},
		Output: InvalidReaction,
	}, {
		Name:   "Multiple Character Strings",
		Input:  []string{"🧖 hello 🦋 world", "😀 hello 😀 world"},
		Output: InvalidReaction,
	}, {
		Name:   "Single normal characters",
		Input:  []string{"A", "b", "1"},
		Output: InvalidReaction,
	}, {
		Name:   "Multiple normal characters",
		Input:  []string{"AA", "bag"},
		Output: InvalidReaction,
	}, {
		Name:   "Multiple normal characters and emojis",
		Input:  []string{"🍆A", "👍😘A"},
		Output: InvalidReaction,
	}, {
		Name:   "No characters",
		Input:  []string{""},
		Output: InvalidReaction,
	},
}

// Unit test of ValidateReaction.
func TestValidateReaction(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			for i, r := range tt.Input {
				err := ValidateReaction(r)
				if err != tt.Output {
					t.Errorf("%2d. Incorrect response for reaction %q %X."+
						"\nexpected: %s\nreceived: %s",
						i, r, []rune(r), tt.Output, err)
				}
			}
		})
	}
}
