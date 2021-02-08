///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package contact

import (
	"bytes"
	"crypto"
	"encoding/base64"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

const sizeByteLength = 2
const fingerprintLength = 15

// Contact implements the Contact interface defined in interface/contact.go,
// in go, the structure is meant to be edited directly, the functions are for
// bindings compatibility
type Contact struct {
	ID             *id.ID
	DhPubKey       *cyclic.Int
	OwnershipProof []byte
	Facts          fact.FactList
}

// Marshal saves the Contact in a compact byte slice. The byte slice has
// the following structure (not to scale).
//
// +----------+----------------+---------+----------+----------+----------------+----------+
// | DhPubKey | OwnershipProof |  Facts  |    ID    |          |                |          |
// |   size   |      size      |   size  |          | DhPubKey | OwnershipProof | FactList |
// |  2 bytes |     2 bytes    | 2 bytes | 33 bytes |          |                |          |
// +----------+----------------+---------+----------+----------+----------------+----------+
func (c Contact) Marshal() []byte {
	var buff bytes.Buffer
	b := make([]byte, sizeByteLength)

	// Write size of DhPubKey
	var dhPubKey []byte
	if c.DhPubKey != nil {
		dhPubKey = c.DhPubKey.BinaryEncode()
		binary.PutVarint(b, int64(len(dhPubKey)))
	}
	buff.Write(b)

	// Write size of OwnershipProof
	binary.PutVarint(b, int64(len(c.OwnershipProof)))
	buff.Write(b)

	// Write length of Facts
	factList := c.Facts.Stringify()
	binary.PutVarint(b, int64(len(factList)))
	buff.Write(b)

	// Write ID
	if c.ID != nil {
		buff.Write(c.ID.Marshal())
	} else {
		// Handle nil ID
		buff.Write(make([]byte, id.ArrIDLen))
	}

	// Write DhPubKey
	buff.Write(dhPubKey)

	// Write OwnershipProof
	buff.Write(c.OwnershipProof)

	// Write fact list
	buff.Write([]byte(factList))

	return buff.Bytes()
}

// Creates a 15 character long fingerprint of contact
// off of the ID and DH public key
func (c Contact) GetFingerprint() string {
	// Generate hash
	sha := crypto.SHA256
	h := sha.New()

	// Hash Id and public key
	h.Write(c.ID.Bytes())
	h.Write(c.DhPubKey.Bytes())
	data := h.Sum(nil)

	// Encode hash and truncate
	return base64.StdEncoding.EncodeToString(data[:])[:fingerprintLength]
}

// Unmarshal decodes the byte slice produced by Contact.Marshal into a Contact.
func Unmarshal(b []byte) (Contact, error) {
	if len(b) < sizeByteLength*3+id.ArrIDLen {
		return Contact{}, errors.Errorf("Length of provided buffer (%d) too "+
			"short; length must be at least %d.",
			len(b), sizeByteLength*3+id.ArrIDLen)
	}

	c := Contact{DhPubKey: &cyclic.Int{}}
	var err error
	buff := bytes.NewBuffer(b)

	// Get size of each field
	dhPubKeySize, _ := binary.Varint(buff.Next(sizeByteLength))
	ownershipProofSize, _ := binary.Varint(buff.Next(sizeByteLength))
	factsSize, _ := binary.Varint(buff.Next(sizeByteLength))

	// Get and unmarshal ID
	c.ID, err = id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return Contact{}, errors.Errorf("Failed to unmarshal Contact ID: %+v", err)
	}

	// Handle nil ID
	if bytes.Equal(c.ID.Marshal(), make([]byte, id.ArrIDLen)) {
		c.ID = nil
	}

	// Get and decode DhPubKey
	if dhPubKeySize == 0 {
		// Handle nil key
		c.DhPubKey = nil
	} else {
		if err = c.DhPubKey.BinaryDecode(buff.Next(int(dhPubKeySize))); err != nil {
			return Contact{}, errors.Errorf("Failed to binary decode Contact DhPubKey: %+v", err)
		}
	}

	// Get OwnershipProof
	if ownershipProofSize == 0 {
		// Handle nil OwnershipProof
		c.OwnershipProof = nil
	} else {
		c.OwnershipProof = buff.Next(int(ownershipProofSize))
	}

	// Get and unstringify fact list
	c.Facts, _, err = fact.UnstringifyFactList(string(buff.Next(int(factsSize))))
	if err != nil {
		return Contact{}, errors.Errorf("Failed to unstringify Contact fact list: %+v", err)
	}

	return c, nil
}
