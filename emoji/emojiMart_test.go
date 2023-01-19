////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	_ "embed"
	"encoding/json"
	"gitlab.com/xx_network/primitives/utils"
	"reflect"
	"testing"
)

//go:embed emojiMart.json
var emojiMartJson []byte

// Tests that marshaling the emojiMartData object and unmarshalling that JSON
// data back into an object does not cause loss in data.
func Test_emojiMartData_JSON_Marshal_Unmarshal(t *testing.T) {
	exampleData := emojiMartSet{
		Categories: []category{
			{ID: "100", Emojis: []emojiID{"100"}},
			{ID: "21"},
			{ID: "20"},
		},
		Emojis: map[emojiID]emoji{
			"100": {
				ID:       "100",
				Name:     "Hundred Points",
				Keywords: []string{"hunna"},
				Skins:    nil,
				Version:  0,
			},
		},
		Aliases: map[string]emojiID{
			"lady_beetle": "ladybug",
		},
		Sheet: map[string]interface{}{
			"test": "data",
		},
	}

	marshaled, err := json.Marshal(&exampleData)
	if err != nil {
		t.Fatalf("Failed to marshal: %+v", err)
	}

	unmarshalData := emojiMartSet{}
	err = json.Unmarshal(marshaled, &unmarshalData)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %+v", err)
	}

	if reflect.DeepEqual(unmarshalData, marshaled) {
		t.Fatalf("Failed to unmarshal example and maintain original data."+
			"\nExpected: %+v"+
			"\nReceived: %+v", exampleData, unmarshalData)
	}
}

// Tests that the example front end JSON can be marshalled into the custom
// emojiMartData object. Also tests the emojiMartData object can be marshalled
// back into JSON without losing data.
func Test_emojiMartDataJSON_Example(t *testing.T) {

	emojiMart := &emojiMartSet{}
	err := json.Unmarshal(emojiMartJson, emojiMart)
	if err != nil {
		t.Fatalf("Failed to unamrshal: %+v", err)
	}

	marshalled, err := json.Marshal(emojiMart)
	if err != nil {
		t.Fatalf("Failed to marshal: %+v", err)
	}

	utils.WriteFileDef("marshalled-EmojiMart.json")

	t.Logf("original: %d\nmarshalled: %d", len(emojiMartJson), len(marshalled))

}
