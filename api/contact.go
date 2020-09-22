////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// Contact implements the Contact interface defined in bindings/interfaces.go,
type Contact struct {
	ID            id.ID
	PubKey        rsa.PublicKey
	Salt          []byte
	Authenticated bool
	Confirmed     bool
}

// GetID returns the user ID for this user.
func (c Contact) GetID() []byte {
	return c.ID.Bytes()
}

// GetPublicKey returns the publickey bytes for this user.
func (c Contact) GetPublicKey() []byte {
	return rsa.CreatePublicKeyPem(&c.PubKey)
}

// GetSalt returns the salt used to initiate an authenticated channel
func (c Contact) GetSalt() []byte {
	return c.Salt
}

// IsAuthenticated returns true if an authenticated channel exists for
// this user so we can begin to send messages.
func (c Contact) IsAuthenticated() bool {
	return c.Authenticated
}

// IsConfirmed returns true if the user has confirmed the authenticated
// channel on their end.
func (c Contact) IsConfirmed() bool {
	return c.Confirmed
}

// Marshal creates a serialized representation of a contact for
// out-of-band contact exchange.
func (c Contact) Marshal() ([]byte, error) {
	return nil, nil
}
