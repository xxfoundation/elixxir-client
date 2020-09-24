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
