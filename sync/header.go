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
	"sync"
)

// headerVersion is the most up-to-date edition of the Header object. If the
// Header object goes through significant restructuring, the version should be
// incremented. It is up to the developer to support older versions.
const headerVersion = 0

// Error constants.
const (
	entryAlreadyExistsErr = "entry for key %s already exists and cannot be overwritten."
	entryDoesNotExistErr  = "entry for key %s could not be found."
)

// Header is header information for a transaction log.
// This will contain a list of entries.
type Header struct {
	// version is the edition of this version. This is defaulted to
	// headerVersion.
	version uint16

	// entries is a list of transactions. Each entry in the header is used to
	// encode things like the encryption for a specific device ID or other
	// metadata as needed.
	entries map[string]string

	mux sync.Mutex
}

// NewHeader is the constructor of a Header object.
func NewHeader() *Header {
	return &Header{
		version: headerVersion,
		entries: make(map[string]string, 0),
	}
}

// Set will write to the entries list the value, if there is not already a value
// for the passed in key.
func (h *Header) Set(key, value string) error {
	h.mux.Lock()
	defer h.mux.Unlock()
	if _, exists := h.entries[key]; exists {
		return errors.Errorf(entryAlreadyExistsErr, key)
	}

	h.entries[key] = value
	return nil
}

// Get will retrieve an entry with the passed in key. If no entry exists for
// this key, an error will be returned.
func (h *Header) Get(key string) (string, error) {
	h.mux.Lock()
	defer h.mux.Unlock()

	value, exists := h.entries[key]
	if !exists {
		return "", errors.Errorf(entryDoesNotExistErr, key)
	}

	return value, nil
}

// header is an object adhering to Header. This serves as the marshal-able
// an unmarshal-able object such that Header may adhere to the json.Marshaler
// and json.Unmarshaler interfaces.
type header struct {
	*Header
}

// MarshalJSON marshals the Header into valid JSON. This function adheres to the
// json.Marshaler interface.
func (h *Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(header{h})
}

// UnmarshalJSON unmarshalls JSON into the cipher. This function adheres to the
// json.Unmarshaler interface.
func (h *Header) UnmarshalJSON(data []byte) error {
	headerData := header{}
	err := json.Unmarshal(data, &headerData)
	if err != nil {
		return err
	}

	*h = Header{
		version: headerData.version,
		entries: headerData.entries,
	}

	return nil
}
