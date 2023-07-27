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
	"gitlab.com/elixxir/wasm-utils/storage"
)

// StateKV is a global that allows switching out the storage backend for
// certain KV operations. Defaults to backed by local storage.
var StateKV = storage.GetLocalStorage()
