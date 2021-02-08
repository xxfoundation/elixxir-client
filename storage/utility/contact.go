///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package utility

import (
	"fmt"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const currentContactVersion = 0

func StoreContact(kv *versioned.KV, c contact.Contact) error {
	now := time.Now()

	obj := versioned.Object{
		Version:   currentContactVersion,
		Timestamp: now,
		Data:      c.Marshal(),
	}

	return kv.Set(makeContactKey(c.ID), &obj)
}

func LoadContact(kv *versioned.KV, cid *id.ID) (contact.Contact, error) {
	vo, err := kv.Get(makeContactKey(cid))
	if err != nil {
		return contact.Contact{}, err
	}

	return contact.Unmarshal(vo.Data)
}

func DeleteContact(kv *versioned.KV, cid *id.ID) error {
	return kv.Delete(makeContactKey(cid))
}

func makeContactKey(cid *id.ID) string {
	return fmt.Sprintf("Contact:%s", cid)
}
