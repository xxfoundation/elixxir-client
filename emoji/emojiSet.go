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
	replacementMap map[codepoint]skin

	// supportedEmojis contains a list of all Unicode codepoints for the emojis
	// that are supported. This allows for quick lookup when comparing against
	// the frontend list of emojis.
	supportedEmojis map[codepoint]struct{}
}

// NewSet constructs a Set.
func NewSet() *Set {
	return &Set{
		replacementMap: map[codepoint]skin{
			"2764-fe0f": {
				Unified: "2764", // Has codepoint "2764-fe0f" in front-end
				Native:  "❤",
			},
		},
		supportedEmojis: emojiListToMap(gomoji.AllEmojis()),
	}
}

// SanitizeFrontEndEmojis will sanitize the list of emojis that front end
// supports. It will be sanitized by modifying the list to determine the union
// of supported emojis between front end (EmojiMart) and backend (gomoji.emojiMap).
func (s *Set) SanitizeFrontEndEmojis(frontendEmojiSetJson []byte) ([]byte, error) {

	// Unmarshal front end's JSON
	var frontEndEmojiSet emojiMartData
	err := json.Unmarshal(frontendEmojiSetJson, &frontEndEmojiSet)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal front end emoji JSON: %+v", err)
	}

	// Find all incompatible emojis in the front end set
	emojisToRemove := s.findIncompatibleEmojis(&frontEndEmojiSet)

	// Remove all incompatible emojis from the set
	removeIncompatibleEmojis(&frontEndEmojiSet, emojisToRemove)

	return json.Marshal(frontEndEmojiSet)
}

// findIncompatibleEmojis returns a list of emojis in the emojiMartData that are
// not supported by the Set. Also, any emojiMartData emoji codepoints that are
// incompatible and have replacements (as defined in Set) are replaced.
func (s *Set) findIncompatibleEmojis(set *emojiMartData) (emojisToRemove []emojiID) {
	// Iterate over all emojis in the emojiMartData.Emojis list
	for id, Emoji := range set.Emojis {
		var newSkins []skin
		for _, Skin := range Emoji.Skins {
			// Determine if the emoji's codepoint should be replaced or removed
			replacement, replace := s.replace(Skin.Unified)
			if replace {
				newSkins = append(newSkins, replacement)
			} else if !s.remove(Skin.Unified) {
				newSkins = append(newSkins, Skin)
			}
		}

		if len(newSkins) > 0 {
			// Write to the set the possible edits (if emojis were replaced
			// or removed)
			Emoji.Skins = newSkins
			set.Emojis[id] = Emoji
		} else {
			// If all skins have been removed, then mark the emoji for removal
			emojisToRemove = append(emojisToRemove, id)
		}
	}

	return emojisToRemove
}

// removeIncompatibleEmojis removes all the emojis in emojisToRemove from the
// emojiMartData set.
func removeIncompatibleEmojis(set *emojiMartData, emojisToRemove []emojiID) {
	// Remove all incompatible emojis from the emojiMartData.Emojis list
	for _, char := range emojisToRemove {
		delete(set.Emojis, char)
	}

	// Remove all incompatible emojis from the emojiMartData.Categories list
	for _, cat := range set.Categories {
		// Iterate over the emoji list backwards to make removal of elements
		// from the slice easier
		for i := len(cat.Emojis) - 1; i >= 0; i-- {
			for _, char := range emojisToRemove {
				if cat.Emojis[i] == char {
					cat.Emojis = append(cat.Emojis[:i], cat.Emojis[i+1:]...)
				}
			}
		}
	}

	// Remove all incompatible emojis from the emojiMartData.Aliases list
	for alias, id := range set.Aliases {
		for _, removedId := range emojisToRemove {
			if id == removedId {
				delete(set.Aliases, alias)
			}
		}
	}
}

// replace returns whether the front end Unicode codepoint must be replaced.
// It will return a boolean on whether this codepoint needs to be replaced
// and what the codepoint must be replaced with.
func (s *Set) replace(code codepoint) (replacement skin, replace bool) {
	replacement, replace = s.replacementMap[code]
	return replacement, replace
}

// remove returns true if the code point should be removed from the parent list.
func (s *Set) remove(code codepoint) bool {
	_, exists := s.supportedEmojis[code]
	return !exists
}

// emojiListToMap constructs a map for simple lookup for gomoji.Emoji's
// Unicode codepoint.
func emojiListToMap(list []gomoji.Emoji) map[codepoint]struct{} {
	emojiMap := make(map[codepoint]struct{}, len(list))
	for _, e := range list {
		emojiMap[backToFrontCodePoint(e.CodePoint)] = struct{}{}
	}
	return emojiMap
}

// backToFrontCodePoint converts Unicode codepoint format found in gomoji.Emoji
// to the one passed in by frontend. The specific conversion is making it
// lowercase and replacing " " with "-".
func backToFrontCodePoint(code string) codepoint {
	return codepoint(strings.ToLower(strings.ReplaceAll(code, " ", "-")))
}
