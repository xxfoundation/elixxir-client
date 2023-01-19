////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

import (
	"embed"
	_ "embed"
	"encoding/json"
	"reflect"
	"testing"
)

// content is used to perform file IO. This allows file operations to be
// performed within tests without introducing incompatibility with
// generating WASM.
//
//go:embed emojiMart.json
var content embed.FS

func Test_emojiMartData_JSON_Marshal_Unmarshal(t *testing.T) {
	exampleData := emojiMartData{
		Categories: []category{
			{
				Id: "100",
				Emojis: []emojiID{
					"100",
				},
			},
			{
				Id: "21",
			},
			{
				Id: "20",
			},
		},
		Emojis: map[emojiID]emoji{
			"100": {
				Id:       "100",
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

	unmarshalData := emojiMartData{}
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

func Test_emojiMartDataJSON_Example(t *testing.T) {
	jsonData, err := content.ReadFile("emojiMart.json")
	if err != nil {
		t.Fatalf("Failed to read emoji-Mart.json: %+v", err)
	}

	emojiMart := &emojiMartData{}
	err = json.Unmarshal(jsonData, emojiMart)
	if err != nil {
		t.Fatalf("Failed to unamrshal: %+v", err)
	}

	if len(emojiMart.Emojis) == 0 {
		t.Fatalf("Did not load front end data as expected")
	}

}
