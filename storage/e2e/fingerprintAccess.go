///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

type fingerprintAccess interface {
	// Receives a list of fingerprints to add. Overrides on collision.
	add([]*Key)
	// Receives a list of fingerprints to delete. Ignores any not available Keys
	remove([]*Key)
}
