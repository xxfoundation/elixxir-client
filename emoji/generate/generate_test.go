////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"gitlab.com/elixxir/client/v4/emoji"
)

var (
	testText = `
# emoji-test.txt
# Date: 2022-08-12, 20:24:39 GMT
# © 2022 Unicode®, Inc.
#
# Emoji Keyboard/Display Test Data for UTS #51
# Version: 15.0
#
# For documentation and usage, see https://www.unicode.org/reports/tr51


# group: Smileys & Emotion

# subgroup: face-smiling
1F600     ; fully-qualified     # 😀 E1.0 grinning face
1F603     ; fully-qualified     # 😃 E0.6 grinning face with big eyes
1F643     ; fully-qualified     # 🙃 E1.0 upside-down face

# subgroup: face-affection
263A FE0F ; fully-qualified     # ☺️ E0.6 smiling face

# group: Animals & Nature

# subgroup: animal-mammal
1F435     ; fully-qualified     # 🐵 E0.6 monkey face
1F412     ; fully-qualified     # 🐒 E0.6 monkey

#EOF`

	testFile = emoji.File{
		Date:    "2022-08-12, 20:24:39 GMT",
		Version: "15.0",
		Map: emoji.Map{
			"😀":  {"😀", "grinning face", "E1.0", "1F600", "Smileys & Emotion", "face-smiling"},
			"😃":  {"😃", "grinning face with big eyes", "E0.6", "1F603", "Smileys & Emotion", "face-smiling"},
			"🙃":  {"🙃", "upside-down face", "E1.0", "1F643", "Smileys & Emotion", "face-smiling"},
			"☺️": {"☺️", "smiling face", "E0.6", "263A FE0F", "Smileys & Emotion", "face-affection"},
			"🐵":  {"🐵", "monkey face", "E0.6", "1F435", "Animals & Nature", "animal-mammal"},
			"🐒":  {"🐒", "monkey", "E0.6", "1F412", "Animals & Nature", "animal-mammal"},
		},
	}
)

// Tests that download can download the expected string from the test server.
func Test_download(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := fmt.Fprintf(w, testText); err != nil {
				t.Fatal(err)
			}
		}),
	)
	defer ts.Close()

	file, _, err := download(ts.URL)
	if err != nil {
		t.Fatalf("error: %+v", err)
	}

	if file != testText {
		t.Errorf("Failed to download expected file.\nexpected: %s\nreceived: %s",
			testText, file)
	}
}

// Tests that Params.parse parses a known text file into its expected
// emoji.File.
func TestParams_parse(t *testing.T) {
	p := DefaultParams()
	f := p.parse(testText)

	if !reflect.DeepEqual(testFile, f) {
		t.Errorf("Parsed emoji file does not match expected."+
			"\nexpected: %+v\nreceived: %+v", testFile, f)
	}
}

// Tests that Params.saveListToJson saves the expected object to file by loading
// it, unmarshalling it, and comparing it to the expected emoji.File.
func TestParams_saveListToJson(t *testing.T) {
	p := DefaultParams()
	p.JsonOutput = "temp.json"
	defer func() {
		if err := os.Remove(p.JsonOutput); err != nil {
			t.Error(err)
		}
	}()

	err := p.saveListToJson(testFile)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(p.JsonOutput)
	if err != nil {
		t.Fatal(err)
	}

	var loadedFile emoji.File
	err = json.Unmarshal(data, &loadedFile)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(testFile, loadedFile) {
		t.Errorf("Unamrshalled emoji file does not match expected."+
			"\nexpected: %+v\nreceived: %+v", testFile, loadedFile)
	}
}
