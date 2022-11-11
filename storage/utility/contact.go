////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"fmt"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentContactVersion = 0

// StoreContact writes a contact into a versioned.KV using the
// contact ID as the key.
//
// Parameters:
//   - kv - the key value store to write the contact.
//   - c - the contact object to store.
//
// Returns:
//   - error if the write fails to succeed for any reason.
func StoreContact(kv *versioned.KV, c contact.Contact) error {
	now := netTime.Now()

	obj := versioned.Object{
		Version:   currentContactVersion,
		Timestamp: now,
		Data:      c.Marshal(),
	}

	return kv.Set(makeContactKey(c.ID), &obj)
}

// LoadContact reads a contact from a versioned.KV vie their contact ID.
//
// Parameters:
//   - kv - the key value store to read the contact
//   - cid - the contacts unique *id.ID to load
//
// Returns:
//   - contact.Contact object populated with the user info, or empty on error.
//   - version number of the contact loaded.
//   - error if an error occurs, or nil otherwise
func LoadContact(kv *versioned.KV, cid *id.ID) (contact.Contact, error) {
	vo, err := kv.Get(makeContactKey(cid), currentContactVersion)
	if err != nil {
		return contact.Contact{}, err
	}

	return contact.Unmarshal(vo.Data)
}

// DeleteContact removes the contact identified by cid from the kv.
//
// Parameters:
//   - kv - the key value store to delete from
//   - cid - the contacts unique *id.ID to delete
//
// Returns:
//   - error if an error occurs or nil otherwise
func DeleteContact(kv *versioned.KV, cid *id.ID) error {
	return kv.Delete(makeContactKey(cid), currentContactVersion)
}

func makeContactKey(cid *id.ID) string {
	return fmt.Sprintf("Contact:%s", cid)
}
