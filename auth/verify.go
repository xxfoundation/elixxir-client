///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage"
	cAuth "gitlab.com/elixxir/crypto/e2e/auth"
)

func VerifyOwnership(received, verified contact.Contact, storage *storage.Session) bool {
	myHistoricalPrivKey := storage.E2e().GetDHPrivateKey()
	return cAuth.VerifyOwnershipProof(myHistoricalPrivKey, verified.DhPubKey,
		storage.E2e().GetGroup(), received.OwnershipProof)
}
