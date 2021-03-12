///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/xx_network/primitives/id"
)

// Contact contains the information to respond to a single-use contact.
type Contact struct {
	partner       *id.ID          // ID of the person to respond to
	partnerPubKey *cyclic.Int     // Public key of the partner
	dhKey         *cyclic.Int     // DH key
	tagFP         singleUse.TagFP // Identifies which callback to use
	maxParts      uint8           // Max number of messages allowed in reply
	used          *int32          // Atomic variable
}

// NewContact initialises a new Contact with the specified fields.
func NewContact(partner *id.ID, partnerPubKey, dhKey *cyclic.Int,
	tagFP singleUse.TagFP, maxParts uint8) Contact {
	used := int32(0)
	return Contact{
		partner:       partner,
		partnerPubKey: partnerPubKey,
		dhKey:         dhKey,
		tagFP:         tagFP,
		maxParts:      maxParts,
		used:          &used,
	}
}

// GetMaxParts returns the maximum number of message parts that can be sent in a
// reply.
func (c Contact) GetMaxParts() uint8 {
	return c.maxParts
}

// GetPartner returns a copy of the partner ID.
func (c Contact) GetPartner() *id.ID {
	return c.partner.DeepCopy()
}

// String returns a string of the Contact structure.
func (c Contact) String() string {
	format := "Contact{partner:%s  partnerPubKey:%s  dhKey:%s  tagFP:%s  maxParts:%d  used:%d}"
	return fmt.Sprintf(format, c.partner, c.partnerPubKey.Text(10),
		c.dhKey.Text(10), c.tagFP, c.maxParts, *c.used)
}

// Equal determines if c and b have equal field values.
func (c Contact) Equal(b Contact) bool {
	return c.partner.Cmp(b.partner) &&
		c.partnerPubKey.Cmp(b.partnerPubKey) == 0 &&
		c.dhKey.Cmp(b.dhKey) == 0 &&
		c.tagFP == b.tagFP &&
		c.maxParts == b.maxParts &&
		*c.used == *b.used
}
