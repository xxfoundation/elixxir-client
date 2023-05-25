////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// state_js.go provides an interface for storing persistent client state
// and its associated implementations. Exclusively for web clients.

package utility

import (
	"os"
	"syscall/js"
)

// StateKV is a global that allows switching out the storage backend for
// certain KV operations. Defaults to backed by local storage.
var StateKV = newLocalStorageState()

// WebState defines an interface for setting persistent state in a KV format
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
