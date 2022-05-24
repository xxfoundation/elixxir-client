///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Session is a sub-interface of the storage.Session interface relevant to
// the methods used in this package.
type Session interface {
	GetTransmissionID() *id.ID
	IsPrecanned() bool
	GetTransmissionRSA() *rsa.PrivateKey
	GetRegistrationTimestamp() time.Time
	GetTransmissionSalt() []byte
	GetTransmissionRegistrationValidationSignature() []byte
}
