////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/v5/e2e"
	"gitlab.com/elixxir/crypto/contact"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
)

// VerifyOwnership calls the cAuth.VerifyOwnershipProof function
// to cryptographically prove the received ownership.
func VerifyOwnership(received, verified contact.Contact, e2e e2e.Handler) bool {
	myHistoricalPrivKey := e2e.GetHistoricalDHPrivkey()
	return cAuth.VerifyOwnershipProof(myHistoricalPrivKey, verified.DhPubKey,
		e2e.GetGroup(), received.OwnershipProof)
}
