////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

// emojiMartData adheres to the JSON file format provided by front end.
// Specifically, this adheres to the emoji-mart library for Typescript.
// The general page is found [here]: https://github.com/missive/emoji-mart/
// while an example JSON document may be found [here]: https://github.com/missive/emoji-mart/blob/main/packages/emoji-mart-data/sets/14/native.json
type emojiMartData struct {
	Categories map[string]interface{} `json:"categories"`
	Emojis     map[string]emoji       `json:"emojis"`
	Sheet      map[string]interface{} `json:"sheet"`
	Aliases    map[string]interface{} `json:"aliases"`
}

// emoji adheres to the emoji field found within the JSON file that is provided
// by the emoji-mart library (see emojiMartData for more detail).
type emoji struct {
	Id       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Skins    []skin   `json:"skins"`
	Version  int      `json:"version"`
}

// skin adheres to the skin field within the emoji field of the JSON file that
// is provided  by the emoji-mart library (see emojiMartData for more detail).
type skin struct {
	Unified string `json:"unified"`
	Native  string `json:"native"`
}
