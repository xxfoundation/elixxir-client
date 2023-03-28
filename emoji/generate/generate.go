////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/pkg/errors"

	"gitlab.com/elixxir/client/v4/emoji"
)

// emojiURL is the URL to the list of the latest emojis published by Unicode for
// testing in keyboards and when displayed/processed. It is parsed to create a
// list of all valid emojis.
const emojiURL = "https://unicode.org/Public/emoji/latest/emoji-test.txt"

// Params contains all the optional parameters for downloading and parsing the
// emoji list and saving the map files.
type Params struct {
	// DownloadURL is the URL where the emoji list is downloaded from.
	DownloadURL string

	// GoOutput is the filepath to save the Go file to. If left empty, no Go
	// file is created.
	GoOutput string

	// JsonOutput is the filepath to save the JSON file to. If left empty, no
	// JSON file is created.
	JsonOutput string

	// CodePointDelim is the separator used between codepoints.
	CodePointDelim string
}

// DefaultParams returns the default configuration for Params.
func DefaultParams() Params {
	return Params{
		DownloadURL:    emojiURL,
		GoOutput:       "./emoji/data.go",
		JsonOutput:     "",
		CodePointDelim: " ",
	}
}

// generate generates the Go and/or JSON file of a map of all emojis.
func generate(p Params) error {
	body, err := download(p.DownloadURL)
	if err != nil {
		return err
	}

	emojis := p.parse(body)

	err = p.saveListToJson(emojis)
	if err != nil {
		return errors.Wrap(err, "failed to save JSON file")
	}

	err = p.saveListToGo(emojis)
	if err != nil {
		return errors.Wrap(err, "failed to save Go file")
	}

	return nil
}

// download downloads and returns the content of the file URL.
func download(fileURL string) (string, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", err
	}

	if resp.StatusCode > 299 {
		return "", errors.Errorf("response failed with status code %d: %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	defer func(Body io.ReadCloser) {
		err2 := Body.Close()
		if err2 != nil {
			err = errors.Wrapf(err, "failed to close body: %+v", err2)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parse parses the emoji-test.txt file into a List and GroupedList.
func (p *Params) parse(s string) emoji.Map {
	emojis := make(emoji.Map)
	s = strings.SplitN(s, "\n\n\n", 2)[1]

	var group, subGroup string
	for _, line := range strings.Split(s, "\n") {
		if len(line) == 0 {
			continue
		} else if line == "#EOF" {
			break
		}

		fields := strings.Fields(line)

		if fields[0] == "#" {
			if len(fields) > 2 {
				switch fields[1] {
				case "group:":
					group = strings.TrimSpace(
						strings.SplitN(line, "# group:", 2)[1])
				case "subgroup:":
					subGroup = strings.TrimSpace(
						strings.SplitN(line, "# subgroup:", 2)[1])
				}
			}
			continue
		}

		var codePoints []string
		for j, codepoint := range fields {
			if codepoint == ";" {
				codePoints = fields[:j]
				fields = fields[j:]
				break
			}
		}

		comment := fields[4]

		e := emoji.Emoji{
			Character: fields[3],
			Name:      strings.TrimSpace(strings.SplitN(line, comment, 2)[1]),
			Comment:   comment,
			CodePoint: strings.Join(codePoints, p.CodePointDelim),
			Group:     group,
			Subgroup:  subGroup,
		}

		emojis[e.Character] = e
	}

	return emojis
}

// saveListToJson saves the emoji list to the JSON output file. If no file is
// set, nothing is saved.
func (p *Params) saveListToJson(emojis emoji.Map) error {
	if p.JsonOutput == "" {
		return nil
	}

	data, err := json.MarshalIndent(emojis, "", "\t")
	if err != nil {
		return err
	}

	return os.WriteFile(p.JsonOutput, data, 0777)
}

// saveListToGo generates a static Go file containing the List.
func (p *Params) saveListToGo(emojis emoji.Map) error {
	if p.GoOutput == "" {
		return nil
	}

	tplFile, err := template.New("EmojisMap").Parse(textTplFileEmojis)
	if err != nil {
		return err
	}

	output, err := os.Create(p.GoOutput)
	if err != nil {
		return err
	}

	return tplFile.Execute(output, emojis)
}

const textTplFileEmojis = `////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package emoji

var emojiMap = Map{ {{ range $index, $val  := . }}
	"{{ $index }}": {
		Character: "{{ $val.Character }}",
		Name:      "{{ $val.Name }}",
		Comment:   "{{ $val.Comment }}",
		CodePoint: "{{ $val.CodePoint }}",
		Group:     "{{ $val.Group }}",
		Subgroup:  "{{ $val.Subgroup }}",
	},{{ end }}
}
`
