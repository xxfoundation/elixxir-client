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
	"reflect"
	"testing"
)

//go:embed emojiMart.json
var emojiMartJson []byte

// Tests that marshaling the emojiMartData object and unmarshalling that JSON
// data back into an object does not cause loss in data.
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

// Tests that the example front end JSON can be marshalled into the custom
// emojiMartData object. Also tests the emojiMartData object can be marshalled
// back into JSON without losing data.
func Test_emojiMartDataJSON_Example(t *testing.T) {

	emojiMart := &emojiMartData{}
	err := json.Unmarshal(emojiMartJson, emojiMart)
	if err != nil {
		t.Fatalf("Failed to unamrshal: %+v", err)
	}

	if len(emojiMart.Emojis) == 0 {
		t.Fatalf("Did not load front end data as expected")
	}

}
