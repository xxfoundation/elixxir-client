///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	session2 "gitlab.com/elixxir/client/e2e/ratchet/partner/session"
)

type fingerprintAccess interface {
	// Receives a list of fingerprints to add. Overrides on collision.
	add([]*session2.Cypher)
	// Receives a list of fingerprints to delete. Ignores any not available Keys
	remove([]*session2.Cypher)
}
