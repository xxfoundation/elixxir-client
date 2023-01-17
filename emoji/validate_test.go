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
	"gitlab.com/xx_network/primitives/utils"
	"reflect"
	"strings"
	"testing"
)

func TestValidateReaction(t *testing.T) {
	tests := []struct {
		input string
		err   error
	}{
		{"😀", nil},              // Single-rune emoji (\u1F600)
		{"👋", nil},              // Single-rune emoji (\u1F44B)
		{"👱‍♂️", nil},           // Four-rune emoji (\u1F471\u200D\u2642\uFE0F)
		{"👋🏿", nil},             // Duel-rune emoji with race modification (\u1F44B\u1F3FF)
		{"😀👋", InvalidReaction}, // Two different single-rune emoji (\u1F600\u1F44B)
		{"😀😀", InvalidReaction}, // Two of the same single-rune emoji (\u1F600\u1F600)
		{"🧖 hello 🦋 world", InvalidReaction},
		{"😀 hello 😀 world", InvalidReaction},
		{"🍆", nil},
		{"😂", nil},
		{"❤", nil},
		{"🤣", nil},
		{"👍", nil},
		{"😭", nil},
		{"🙏", nil},
		{"😘", nil},
		{"🥰", nil},
		{"😍", nil},
		{"😊", nil},
		{"☺", nil},
		{"A", InvalidReaction},
		{"b", InvalidReaction},
		{"AA", InvalidReaction},
		{"1", InvalidReaction},
		{"🍆🍆", InvalidReaction},
		{"🍆A", InvalidReaction},
		{"👍👍👍", InvalidReaction},
		{"👍😘A", InvalidReaction},
		{"🧏‍♀️", nil},
		{"❤️", nil},
		{"❤", nil},
	}

	for i, r := range tests {
		err := ValidateReaction(r.input)

		if err != r.err {
			t.Errorf("%2d. Incorrect response for reaction %q %X."+
				"\nexpected: %s\nreceived: %s",
				i, r.input, []rune(r.input), r.err, err)
		}
	}
}

type Skins struct {
	Unified string
	Native  string
}

type Parent struct {
	Skin []Skins
}

func TestFrontEnd_BackEnd_Emojis(t *testing.T) {
	//_, err := jsonquery.LoadURL("https://github.com/missive/emoji-mart/blob/main/packages/emoji-mart-data/sets/11/native.json")
	//if err != nil {
	//	t.Fatalf("Failed to load URL: %+v", err)
	//}

	// single missing example is pretty notable:
	// in front end
	//"heart": {
	//	"id": "heart",
	//		"name": "Red Heart",
	//		"emoticons": [
	//"\u003c3"
	//],
	//"keywords": [
	//"love",
	//"like",
	//"valentines"
	//],
	//"skins": [
	//{
	//"unified": "2764-fe0f",
	//"native": "❤️"
	//}
	//],
	//"version": 1
	//},
	// in Golang, it is simply
	// 		"❤️": {
	//			Slug:        "red-heart",
	//			Character:   "❤",
	//			UnicodeName: "E0.6 red heart",
	//			CodePoint:   "2764",
	// Since this is one example, hard-coding this as an exception is not difficult.
	// This can probably be what is provided by backend to frontend.
	// Notable, is that gomoji having this in its library, passing ❤️ to ValidateEmoji fails.

	allBackendEmojis := gomoji.AllEmojis()
	backendEmojis := make(map[string]bool, 0)
	counter := 0
	for _, emoji := range allBackendEmojis {
		newCodePoint := strings.ToLower(strings.ReplaceAll(emoji.CodePoint, " ", "-"))
		t.Logf("skin code from backend end: \"%s\"", newCodePoint)
		counter += 1
		if backendEmojis[newCodePoint] {
		}
		backendEmojis[newCodePoint] = true
	}

	t.Logf("backendCount: %d", counter)

	t.Logf("number of backend emojis: %d", len(backendEmojis))
	frontEndEmojiData, err := readFrontEnd()
	if err != nil {
		t.Fatal(err)
	}

	//t.Logf("%+v", frontEndEmojiData)

	v := reflect.ValueOf(frontEndEmojiData.Emojis)

	counter = 0
	goCounter := 0
	frontEndEmojis := make(map[string]bool, 0)
	for i := 0; i < v.NumField(); i++ {
		topField := v.Field(i)
		skinsField := topField.FieldByName("Skins")
		if !skinsField.IsZero() {
			convertedSkins := skinsField.Interface().([]struct {
				Unified string `json:"unified"`
				Native  string `json:"native"`
			})

			for _, skin := range convertedSkins {
				counter += 1
				frontEndEmojis[skin.Unified] = true
				//t.Logf("skin code from front end: %s", skin.Unified)
				if backendEmojis[skin.Unified] {
					//t.Logf("skin unified: %s", skin.Unified)

					goCounter += 1
				}
			}

		}

		//listOfFields := reflect.VisibleFields(v.Field(i).Type())
		//for j, field := range listOfFields {
		//	if field.Name == "Skins" {
		//		t.Logf("%+v", v.Field(j))
		//		//t.Logf("%v", field.Type)
		//		//field.Type.Field(0)
		//	}
		//}
	}

	t.Logf("frontend emojis: %d", counter)
	t.Logf("in Common: %d", goCounter)
	t.Logf("frontend list: %d", len(frontEndEmojis))

	for frontEndEmoji := range frontEndEmojis {
		if !backendEmojis[frontEndEmoji] {
			t.Logf("missing: %s", frontEndEmoji)
		}
	}
	//test := v.Field(0)
	//
	//listOfFields := reflect.VisibleFields(test.Type())
	//field := listOfFields[3].Type.Elem()
	//t.Logf("%v", field)

	//writeIndent(frontEndEmojiData, t)

	//for _, cat := range frontEndEmojiData.Categories {
	//	for _, emoji := range cat.Emojis {
	//
	//	}
	//}
	//

	//
	//codepointJson, err := json.Marshal(listOfCodepoints)
	//if err != nil {
	//	t.Fatalf("Failed to marshal map of codepoints: %s", err)
	//}
	//
	//err = utils.WriteFileDef("./emojis-backedn.json", codepointJson)
	//if err != nil {
	//	t.Fatalf("Failed to write code points JSON to file: %+v", err)
	//}
	////

}

func readFrontEnd() (*EmojiMartData, error) {
	frontEndFile := "./emojis-frontend.json"
	frontEndEmojiDataJson, err := utils.ReadFile(frontEndFile)
	if err != nil {
		return nil, errors.Errorf("Failed to read file: %s", err)
	}

	frontEndEmojiData := &EmojiMartData{}
	err = json.Unmarshal(frontEndEmojiDataJson, frontEndEmojiData)
	if err != nil {
		return nil, errors.Errorf("Failed to unmarshal front end emoji data: %s", err)
	}

	return frontEndEmojiData, nil
}

func writeIndent(frontEndEmojiData EmojiMartData) error {
	indentedFrontEnd, err := json.MarshalIndent(frontEndEmojiData, "", "\t")
	if err != nil {
		return errors.Errorf("Failed to marshal indent: %+v", err)
	}

	err = utils.WriteFileDef("./emojis-frontend-indent.json", indentedFrontEnd)
	if err != nil {
		return errors.Errorf("Failed to write to file: %+v", err)
	}

	return err
}
