///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
)

func VerifyOwnership(received, verified contact.Contact, e2e e2e.Handler) bool {
	myHistoricalPrivKey := e2e.GetHistoricalDHPrivkey()
	return cAuth.VerifyOwnershipProof(myHistoricalPrivKey, verified.DhPubKey,
		e2e.GetGroup(), received.OwnershipProof)
}
