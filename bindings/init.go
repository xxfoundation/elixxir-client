///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import "C"

import (
	jww "github.com/spf13/jwalterweatherman"
)

//export Init
func Init() {
	jww.INFO.Printf("cMix bindings loaded...%s\t%s",
		GetVersion(), GetGitVersion())
}
