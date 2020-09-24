////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import jww "github.com/spf13/jwalterweatherman"

import (
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
)

// GetUser returns the current user Identity for this client. This
// can be serialized into a byte stream for out-of-band sharing.
func (c *Client) GetUser() (Contact, error) {
	jww.INFO.Printf("GetUser()")
	return Contact{}, nil
}

// MakeContact creates a contact from a byte stream (i.e., unmarshal's a
// Contact object), allowing out-of-band import of identities.
func (c *Client) MakeContact(contactBytes []byte) (Contact, error) {
	jww.INFO.Printf("MakeContact(%s)", contactBytes)
	return Contact{}, nil
}

// GetContact returns a Contact object for the given user id, or
// an error
func (c *Client) GetContact(uid []byte) (Contact, error) {
	jww.INFO.Printf("GetContact(%s)", uid)
	return Contact{}, nil
}


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
