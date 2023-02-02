////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"github.com/pkg/errors"
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

// header is an object strictly adhering to Header. This serves as the
// marshal-able an unmarshal-able object such that Header may adhere to the
// json.Marshaler and json.Unmarshaler interfaces.
//
// WARNING: If Header is modified, header should reflect these changes to ensure
// no data is lost when calling the json.Marshaler or json.Unmarshaler.
type header struct {
	Version uint16            `json:"version"`
	Entries map[string]string `json:"entries"`
}

// MarshalJSON marshals the Header into valid JSON. This function adheres to the
// json.Marshaler interface.
func (h *Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(header(*h))
}

// UnmarshalJSON unmarshalls JSON into the Header. This function adheres to the
// json.Unmarshaler interface.
func (h *Header) UnmarshalJSON(data []byte) error {
	headerData := header{}
	if err := json.Unmarshal(data, &headerData); err != nil {
		return err
	}
	*h = Header(headerData)
	return nil
}
