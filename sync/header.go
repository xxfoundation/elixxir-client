////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"strings"
)

// headerVersion is the most up-to-date edition of the Header object. If the
// Header object goes through significant restructuring, the version should be
// incremented. It is up to the developer to support older versions.
const headerVersion = 0

// Error constants.
const (
	entryDoesNotExistErr = "entry for key %s could not be found."
)

// Header is header information for a transaction log. It inherits the header
// object to prevent recursive calls by json.Marshal on Header.MarshalJSON.
// Any changes to the Header object fields should be done in header.
type Header header

// NewHeader is the constructor of a Header object.
func NewHeader() *Header {
	return &Header{
		Version: headerVersion,
		Entries: make(map[string]string, 0),
	}
}

// Set will write to the entries list the value, if there is not already a value
// for the passed in key.
func (h *Header) Set(key, value string) error {
	h.Entries[key] = value
	return nil
}

// Get will retrieve an entry with the passed in key. If no entry exists for
// this key, an error will be returned.
func (h *Header) Get(key string) (string, error) {
	value, exists := h.Entries[key]
	if !exists {
		return "", errors.Errorf(entryDoesNotExistErr, key)
	}

	return value, nil
}

// header is the object to which Header strictly adheres. header serves as the
// marshal-able an unmarshal-able object that Header.MarshalJSON and
// Header.UnmarshalJSON utilizes when calling json.Marshal/json.Unmarshal.
//
// WARNING: Modifying header will modify Header, be mindful of the
// consumers when modifying this structure.
type header struct {
	Version uint16            `json:"version"`
	Entries map[string]string `json:"entries"`
}

// serialize serializes a Header object.
//
// Use deserializeHeader to reverse this operation.
func (h *Header) serialize() ([]byte, error) {

	// Marshal header into JSON
	headerMarshal, err := json.Marshal(h)
	if err != nil {
		return nil, err
	}

	// Construct header info
	headerInfo := xxdkTxLogHeader + base64.URLEncoding.EncodeToString(headerMarshal)

	return []byte(headerInfo), nil
}

// deserializeHeader will deserialize header byte data.
//
// This is the inverse operation of Header.serialize.
func deserializeHeader(headerSerial []byte) (*Header, error) {
	// Extract the header
	splitter := strings.Split(string(headerSerial), xxdkTxLogHeader)
	if len(splitter) != 2 {
		// todo: error constant
		return nil, errors.Errorf("unexpected data in serialized header.")
	}

	// Decode transaction
	headerInfo, err := base64.URLEncoding.DecodeString(splitter[1])
	if err != nil {
		return nil, err
	}

	// Unmarshal header
	hdr := &Header{}
	if err = json.Unmarshal(headerInfo, hdr); err != nil {
		return nil, err
	}

	return hdr, nil
}
