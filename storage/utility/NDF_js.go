////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/ndf"
	"os"
	"syscall/js"
)

const NdfStorageKeyNamePrefix = "ndfStorageKey/"

// stateKV is a global that allows switching out the storage backend for
// certain KV operations. Defaults to backed by local storage.
var stateKV = newLocalStorageState()

func LoadNDF(_ *versioned.KV, key string) (*ndf.NetworkDefinition, error) {
	value, err := stateKV.Get(NdfStorageKeyNamePrefix + key)
	if err != nil {
		return nil, err
	}

	return ndf.Unmarshal(value)
}

func SaveNDF(_ *versioned.KV, key string, ndf *ndf.NetworkDefinition) error {
	marshaled, err := ndf.Marshal()
	if err != nil {
		return err
	}

	return stateKV.Set(NdfStorageKeyNamePrefix+key, marshaled)
}

// WebState defines an interface for setting permanent state in a KV format
// specifically for web-based implementations.
type WebState interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

// localStorageState implements WebState backed by local storage.
type localStorageState struct {
	kv js.Value
}

func newLocalStorageState() WebState {
	return &localStorageState{kv: js.Global().Get("localStorage")}
}

func (l *localStorageState) Get(key string) ([]byte, error) {
	value := l.kv.Call("getItem", key)
	if value.IsNull() {
		return nil, os.ErrNotExist
	}
	return []byte(value.String()), nil
}

func (l *localStorageState) Set(key string, value []byte) error {
	l.kv.Call("setItem", key, string(value))
	return nil
}
