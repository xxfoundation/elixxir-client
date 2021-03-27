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
	"github.com/skip2/go-qrcode"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

const (
	version           = "0"
	headTag           = "<xxc"
	footTag           = "xxc>"
	openVerTag        = "("
	closeVerTag       = ")"
	sizeLength        = 2
	minLength         = (sizeLength * 3) + len(headTag) + len(footTag) + id.ArrIDLen
	fingerprintLength = 15
)

// Contact implements the Contact interface defined in interface/contact.go,
// in go, the structure is meant to be edited directly, the functions are for
// bindings compatibility.
type Contact struct {
	ID             *id.ID
	DhPubKey       *cyclic.Int
	OwnershipProof []byte
	Facts          fact.FactList
}

// Marshal saves the Contact in a compact binary format with base 64 encoding.
// The data has a header and footer that specify the format version and allow
// the data to be recognized in a stream of data. The format has the following
// structure.
//
// +----------------+---------------------------------------------------------------------------------------+--------+
// |     header     |                                     contact data                                      | footer |
// +------+---------+----------+----------------+---------+----------+----------+----------------+----------+--------+
// | Open |         | DhPubKey | OwnershipProof |  Facts  |    ID    |          |                |          | Close  |
// | Tag  | Version |   size   |      size      |   size  |          | DhPubKey | OwnershipProof | FactList |  Tag   |
// |      |         |  2 bytes |     2 bytes    | 2 bytes | 33 bytes |          |                |          |        |
// +------+---------+----------+----------------+---------+----------+----------+----------------+----------+--------+
// |     string     |                                    base 64 encoded                                    | string |
// +----------------+---------------------------------------------------------------------------------------+--------+
func (c Contact) Marshal() []byte {
	var buff bytes.Buffer
	b := make([]byte, sizeLength)

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

	// Base 64 encode buffer
	encodedBuff := make([]byte, base64.StdEncoding.EncodedLen(buff.Len()))
	base64.StdEncoding.Encode(encodedBuff, buff.Bytes())

	// Add header tag, version number, and footer tag
	encodedBuff = append([]byte(headTag+openVerTag+version+closeVerTag), encodedBuff...)
	encodedBuff = append(encodedBuff, []byte(footTag)...)

	return encodedBuff
}

// Unmarshal decodes the byte slice produced by Contact.Marshal into a Contact.
func Unmarshal(b []byte) (Contact, error) {
	if len(b) < minLength {
		return Contact{}, errors.Errorf("Length of provided buffer (%d) too "+
			"short; length must be at least %d.",
			len(b), minLength)
	}

	var err error

	// Get data from between the header and footer tags
	b, err = getTagContents(b, headTag, footTag)
	if err != nil {
		return Contact{}, errors.Errorf("data not found: %+v", err)
	}

	// Check that the version matches
	currentVersion, err := getTagContents(b, openVerTag, closeVerTag)
	if string(currentVersion) != version {
		return Contact{}, errors.Errorf("found version %s incomptible, "+
			"requires version %s", string(currentVersion), version)
	}

	// Strip version number
	b = b[len(currentVersion)+len(openVerTag)+len(closeVerTag):]

	// Create new decoder
	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(b))

	// Create a new buffer from the data found between the open and close tags
	var buff bytes.Buffer
	_, err = buff.ReadFrom(decoder)
	if err != nil {
		return Contact{}, errors.Errorf("failed to read from decoder: %+v", err)
	}

	// Get size of each field
	dhPubKeySize, _ := binary.Varint(buff.Next(sizeLength))
	ownershipProofSize, _ := binary.Varint(buff.Next(sizeLength))
	factsSize, _ := binary.Varint(buff.Next(sizeLength))

	// Create empty client
	c := Contact{DhPubKey: &cyclic.Int{}}

	// Get and unmarshal ID
	c.ID, err = id.Unmarshal(buff.Next(id.ArrIDLen))
	if err != nil {
		return Contact{}, errors.Errorf("failed to unmarshal Contact ID: %+v", err)
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
			return Contact{}, errors.Errorf("failed to binary decode Contact DhPubKey: %+v", err)
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
		return Contact{}, errors.Errorf("failed to unstringify Contact fact list: %+v", err)
	}

	return c, nil
}

// GetFingerprint creates a 15 character long fingerprint of the contact off of
// the ID and DH public key.
func (c Contact) GetFingerprint() string {
	// Generate hash
	sha := crypto.SHA256
	h := sha.New()

	// Hash ID and DH public key
	h.Write(c.ID.Bytes())
	h.Write(c.DhPubKey.Bytes())
	data := h.Sum(nil)

	// Base64 encode hash and truncate it
	return base64.StdEncoding.EncodeToString(data)[:fingerprintLength]
}

// MakeQR generates a QR code PNG of the Contact.
func (c Contact) MakeQR(size int, level qrcode.RecoveryLevel) ([]byte, error) {
	qrCode, err := qrcode.Encode(string(c.Marshal()), level, size)
	if err != nil {
		return nil, errors.Errorf("failed to encode contact to QR code: %v", err)
	}

	return qrCode, nil
}

func (c Contact) String() string {
	return "ID: " + c.ID.String() +
		"  DhPubKey: " + c.DhPubKey.Text(10) +
		"  OwnershipProof: " + base64.StdEncoding.EncodeToString(c.OwnershipProof) +
		"  Facts: " + c.Facts.Stringify()
}

// Equal determines if the two contacts have the same values.
func Equal(a, b Contact) bool {
	return a.ID.Cmp(b.ID) &&
		a.DhPubKey.Cmp(b.DhPubKey) == 0 &&
		bytes.Equal(a.OwnershipProof, b.OwnershipProof) &&
		a.Facts.Stringify() == b.Facts.Stringify()
}

// getTagContents returns the bytes between the two tags. An error is returned
// if one ore more tags cannot be found or closing tag precedes the opening tag.
func getTagContents(b []byte, openTag, closeTag string) ([]byte, error) {
	// Search for opening tag
	openIndex := strings.Index(string(b), openTag)
	if openIndex < 0 {
		return nil, errors.New("missing opening tag")
	}

	// Search for closing tag
	closeIndex := strings.Index(string(b), closeTag)
	if closeIndex < 0 {
		return nil, errors.New("missing closing tag")
	}

	// Return an error if the closing tag comes first
	if openIndex > closeIndex {
		return nil, errors.New("tags in wrong order")
	}

	return b[openIndex+len(openTag) : closeIndex], nil
}
