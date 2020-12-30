///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package contact

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

const sizeByteLength = 2

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
		emptyID := make([]byte, id.ArrIDLen)
		buff.Write(emptyID)
	}

	// Write DhPubKey
	buff.Write(dhPubKey)

	// Write OwnershipProof
	buff.Write(c.OwnershipProof)

	// Write fact list
	buff.Write([]byte(factList))

	return buff.Bytes()
}

// Unmarshal decodes the byte slice produced by Contact.Marshal into a Contact.
func Unmarshal(b []byte) (Contact, error) {
	c := Contact{DhPubKey: &cyclic.Int{}}
	var err error
	buff := bytes.NewBuffer(b)

	// Get size (in bytes) of each field
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
		err = c.DhPubKey.BinaryDecode(buff.Next(int(dhPubKeySize)))
		if err != nil {
			return Contact{}, errors.Errorf("Failed to binary decode Contact DhPubKey: %+v", err)
		}
	}

	// Get OwnershipProof
	c.OwnershipProof = buff.Next(int(ownershipProofSize))
	if len(c.OwnershipProof) == 0 {
		// Handle nil OwnershipProof
		c.OwnershipProof = nil
	}

	// Get and unstringify fact list
	c.Facts, _, err = fact.UnstringifyFactList(string(buff.Next(int(factsSize))))
	if err != nil {
		return Contact{}, errors.Errorf("Failed to unstringify Fact List: %+v", err)
	}

	return c, nil
}
