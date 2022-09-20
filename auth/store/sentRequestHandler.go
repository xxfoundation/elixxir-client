////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// SentRequestHandler allows the lower level to assign and remove services
type SentRequestHandler interface {
	Add(sr SentRequestInterface)
	AddLegacySIDH(sr SentRequestInterface)
	Delete(sr SentRequestInterface)
}

// SentRequestInterface defines required functions for SentRequestHandler
type SentRequestInterface interface {
	GetPartner() *id.ID
	GetFingerprint() format.Fingerprint
	IsReset() bool
	// NOTE: Might make sense to just provide the encrypt/decrypt
	// instead here..
	GetMyPubKey() *cyclic.Int
	GetMyPrivKey() *cyclic.Int

	GetPartnerHistoricalPubKey() *cyclic.Int
	GetMyPQPrivateKey() nike.PrivateKey
}
