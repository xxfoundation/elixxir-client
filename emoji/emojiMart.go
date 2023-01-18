////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

// emojiID represents the alias for an emoji. For example, the alias for
// the emoji 💯 would be "100". This adheres strictly to how emoji-mart
// categorizes their emojis within the categories section of their
// JSON file.
type emojiID string

// codepoint represents the Unicode codepoint for an emoji.
// For example, the emoji 💯 would have the codepoint "1f4af".
type codepoint string

// emojiMartData adheres to the JSON file format provided by front end.
// Specifically, this adheres to the emoji-mart library for Typescript.
//
// Doc: https://github.com/missive/emoji-mart/
// JSON example: https://github.com/missive/emoji-mart/blob/main/packages/emoji-mart-data/sets/14/native.json
type emojiMartData struct {
	Categories []category             `json:"categories"`
	Emojis     map[emojiID]emoji      `json:"emojis"`
	Aliases    map[string]emojiID     `json:"aliases"`
	Sheet      map[string]interface{} `json:"sheet"`
}

// category adheres to the category field within the JSON file that is provided
// by the emoji-mart library (see emojiMartData for more detail).
type category struct {
	Emojis []emojiID `json:"emojis"`
	Id     string    `json:"id"`
}

// emoji adheres to the emoji field found within the JSON file that is provided
// by the emoji-mart library (see emojiMartData for more detail).
type emoji struct {
	Id       emojiID  `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Skins    []skin   `json:"skins"`
	Version  float32  `json:"version"`
}

// skin adheres to the skin field within the emoji field of the JSON file that
// is provided  by the emoji-mart library (see emojiMartData for more detail).
type skin struct {
	Unified codepoint `json:"unified"`
	Native  string    `json:"native"`
}
