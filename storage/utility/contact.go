///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"fmt"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentContactVersion = 1

// StoreContact writes a contact into a versioned.KV using the
// contact ID as the key.
//
// Parameters:
//   * kv - the key value store to write the contact.
//   * c - the contact object to store.
// Returns:
//   * error if the write fails to succeed for any reason.
func StoreContact(kv *versioned.KV, c contact.Contact) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentContactVersion,
		Timestamp: now,
		Data:      c.Marshal(),
	}

	return kv.Set(makeContactKey(c.ID), currentContactVersion, &obj)
}

// LoadContact reads a contact from a versioned.KV vie their contact ID.
//
// Parameters:
//   * kv - the key value store to read the contact
//   * cid - the contacts unique *id.ID to load
// Returns:
//   * contact.Contact object populated with the user info, or empty on error.
//   * version number of the contact loaded.
//   * error if an error occurs, or nil otherwise
func LoadContact(kv *versioned.KV, cid *id.ID) (contact.Contact, error) {
	vo, err := kv.Get(makeContactKey(cid), currentContactVersion)
	if err != nil {
		c, contactErr := loadLegacyContactV0(kv, cid)
		// If we got an error on loading legacy contact, and it's not
		// a missing object error, then return a wrapped error
		if contactErr != nil && ekv.Exists(contactErr) {
			return c, errors.Wrapf(err, "%+v", contactErr)
		} else if contactErr != nil && !ekv.Exists(contactErr) {
			return contact.Contact{}, err
		}
		return c, nil
	}

	return contact.Unmarshal(vo.Data)
}

// DeleteContact removes the contact identified by cid from the kv.
//
// Parameters:
//   - kv - the key value store to delete from
//   - cid - the contacts unique *id.ID to delete
// Returns:
//   - error if an error occurs or nil otherwise
func DeleteContact(kv *versioned.KV, cid *id.ID) error {
	return kv.Delete(makeContactKey(cid), currentContactVersion)
}

func makeContactKey(cid *id.ID) string {
	return fmt.Sprintf("Contact:%s", cid)
}

func loadLegacyContactV0(kv *versioned.KV, cid *id.ID) (contact.Contact,
	error) {
	vo, err := kv.Get(makeContactKey(cid), 0)
	if err != nil {
		return contact.Contact{}, err
	}

	jww.DEBUG.Printf("Upgrading legacy V0 contact...")

	c, err := contact.Unmarshal(vo.Data)
	if err != nil {
		return contact.Contact{}, err
	}

	err = StoreContact(kv, c)
	if err != nil {
		return contact.Contact{}, err
	}

	kv.Delete(makeContactKey(cid), 0)
	return c, err
}
