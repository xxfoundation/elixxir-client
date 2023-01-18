////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"encoding/json"
	"github.com/forPelevin/gomoji"
	"github.com/pkg/errors"
	"strings"
)

// Set contains the set of emoji's that the backend supports. This object will
// be used to sanitize the list of emojis front end may support.
type Set struct {
	// replacementMap contains a list of emoji code-points in the front end list
	// that must be replaced to adhere to backend recognized code-points.
	replacementMap map[string]string

	// backendEmojiList contains a list of all Unicode codepoints for the emojis
	// within the gomoji.emojiMap. This allows for quick lookup when comparing
	// against the frontend list of emojis.
	backendEmojiList map[string]struct{}
}

// NewSet constructs an EmojiSet.
func NewSet() *Set {
	return &Set{
		replacementMap: map[string]string{
			"❤️": "❤", // In Unicode: "\u2764\ufe0f":"\u2764"
		},
		backendEmojiList: emojiListToMap(gomoji.AllEmojis()),
	}
}

// SanitizeFrontendEmojiList will sanitize the list of emojis that front end
// supports. It will be sanitized by modifying the list to determine the union
// of supported emojis between front end (EmojiMart) and backend (gomoji.emojiMap).
func (es *Set) SanitizeFrontendEmojiList(frontendEmojiSetJson []byte) ([]byte, error) {

	// Unmarshal front end's JSON
	var frontEndEmojiSet emojiMartData
	err := json.Unmarshal(frontendEmojiSetJson, &frontEndEmojiSet)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal front end emoji JSON: %+v", err)
	}

	// Go through all emojis front end supports and determine
	// what can be supported on the backend
	var emojisToRemove []string
	for char, emoji := range frontEndEmojiSet.Emojis {

		var newSkins []skin
		for _, skin := range emoji.Skins {
			// Determine if we must replace or remove an emoji
			replacement, replace := es.replace(skin.Unified)
			if replace {
				newSkins = append(newSkins, skin{replacement, skin.Native})
			} else if !es.remove(skin.Unified) {
				newSkins = append(newSkins, skin)
			}
		}

		if len(newSkins) > 0 {
			// Write to the set the possible edits (possible no edits were made)
			emoji.Skins = newSkins
			frontEndEmojiSet.Emojis[char] = emoji
		} else {
			emojisToRemove = append(emojisToRemove, char)
		}
	}

	// This will remove from the front end set any emojis that does not exist
	// within the backend set.
	for _, char := range emojisToRemove {
		delete(frontEndEmojiSet.Emojis, char)
	}

	return json.Marshal(frontEndEmojiSet)
}

// replace returns whether the front end Unicode codepoint must be replaced.
// It will return a boolean on whether this codepoint needs to be replaced
// and what the codepoint must be replaced with.
func (es *Set) replace(codePoint string) (replacement string, replace bool) {
	replacement, replace = es.replacementMap[codePoint]
	return replacement, replace
}

// remove returns true if the code point should be removed from the parent list.
func (es *Set) remove(codePoint string) bool {
	_, exists := es.backendEmojiList[codePoint]
	return !exists
}

// emojiListToMap constructs a map for simple lookup for gomoji.Emoji's
// Unicode codepoint.
func emojiListToMap(list []gomoji.Emoji) map[string]struct{} {
	emojiMap := make(map[string]struct{}, len(list))
	for _, emoji := range list {
		emojiMap[backToFrontCodePoint(emoji.CodePoint)] = struct{}{}
	}
	return emojiMap
}

// backToFrontCodePoint converts Unicode codepoint format found in gomoji.Emoji
// to the one passed in by frontend. The specific conversion is making it
// lowercase and replacing " " with "-".
func backToFrontCodePoint(codePoint string) string {
	return strings.ToLower(strings.ReplaceAll(codePoint, " ", "-"))
}
