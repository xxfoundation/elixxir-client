///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type context struct {
	fa fingerprintAccess

	grp *cyclic.Group

	rng *fastRNG.StreamGenerator

	myID *id.ID
}
