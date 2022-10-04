////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"time"
)

// NameService is an interface which encapsulates
// the user identity channel tracking service.
// NameService is currently unused
type NameService interface {

	// GetUsername returns the username.
	GetUsername() string

	// GetChannelValidationSignature returns the validation
	// signature and the time it was signed.
	GetChannelValidationSignature() ([]byte, time.Time)

	// GetChannelPubkey returns the user's public key.
	GetChannelPubkey() ed25519.PublicKey

	// SignChannelMessage returns the signature of the
	// given message.
	SignChannelMessage(message []byte) (signature []byte, err error)

	// ValidateChannelMessage validates that a received channel message's
	// username lease is signed by the NameService
	ValidateChannelMessage(username string, lease time.Time,
		pubKey ed25519.PublicKey, authorIDSignature []byte) bool
}
