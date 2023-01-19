////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import "gitlab.com/elixxir/crypto/channel"

// These objects are imported so that doc linking on pkg.go.dev does not require
// the entire package URL.
var (
	_ = channel.Identity{}
	_ = channel.PrivateIdentity{}
)
