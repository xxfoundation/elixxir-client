///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/crypto/contact"
	"sync"
)

type RequestType uint

const (
	Sent    RequestType = 1
	Receive RequestType = 2
)

type request struct {
	rt RequestType

	// Data if sent
	sent *SentRequest

	// Data if receive
	receive *contact.Contact

	//sidHPublic key of partner
	theirSidHPubKeyA *sidh.PublicKey

	// mux to ensure there is not concurrent access
	mux sync.Mutex
}

type requestDisk struct {
	T  uint
	ID []byte
}
