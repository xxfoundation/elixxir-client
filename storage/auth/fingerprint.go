///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import "gitlab.com/elixxir/crypto/cyclic"

type FingerprintType uint

const (
	General  FingerprintType = 1
	Specific FingerprintType = 2
)

type fingerprint struct {
	Type FingerprintType

	// Only populated if it is general
	PrivKey *cyclic.Int

	// Only populated if it is specific
	Request *ReceivedRequest
}
