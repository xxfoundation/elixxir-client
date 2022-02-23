///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxmutils

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/elixxir/client/bindings"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// RestoreContactsUpdater interface provides a callback funciton
// for receiving update information from RestoreContactsFromBackup.
type RestoreContactsUpdater interface {
	// RestoreContactsCallback is called to report the current # of contacts
	// that are done processing against the total number that need to be
	// processed. If an error occurs it it set on the err variable as a
	// plain string.
	RestoreContactsCallback(current, total int, err string)
}

// RestoreContactsFromBackup takes as input the jason output of the
// `NewClientFromBackup` function, unmarshals it into IDs, looks up
// each ID in user discovery, and initiates a session reset request.
// This function will not return until every id in the list has been sent a
// request. It should be called again and again until it completes.
func RestoreContactsFromBackup(backupPartnerIDs []byte, client *bindings.Client,
	udManager *bindings.UserDiscovery,
	updatesCb RestoreContactsUpdater) error {
	api := client.GetInternalClient()

	// Unmarshal IDs
	var idList []*id.ID
	if err := json.Unmarshal(backupPartnerIDs, &idList); err != nil {
		return err
	}

	var idsToLookup []*id.ID
	for id := range idList {
		// TODO: Check storage for ID, if present, skip
		// if !IDRestored(id) {
		idsToLookup := append(idsToLookup, id)
		//}
	}
	idList = idsToLookup

	// Start looking up contacts with user discovery and feed this
	// contacts channel.
	contactsCh := LookupContacts(idList, client, udManager)

	// Send a reset for each contact we looked up
	cnt := 0
	total := len(idList)
	msg := "Restored from backup"
	me := api.GetUser()
	for contact := range contactsCh {
		cnt += 1
		// Report lookup failures
		if contact.DhPubKey == nil {
			updatesCb.RestoreContactsCallback(cnt, total,
				fmt.Sprintf("ID %s could not be found in "+
					" User Discovery.", contact.ID))
			continue
		}
		_, err := api.ResetSession(contact, me, msg)
		if err != nil {
			jww.WARN.Printf("Could not reset: %v", err)
			// TODO: Add contact object back into channel?
			//       other retry logic?
			continue
		}

		// TODO: Mark ID done in storage
	}

	return nil
}

type lookupcb struct {
	contactsCh chan *contact.Contact
}

func (l lookupcb) Callback(contact *bindings.Contact, err string) {
	if err != nil && err != "" {
		jww.WARN.Printf("Restoring contact: %s", err)
	}
	l.contactsCh <- contact
}

// LookupContacts starts a thread that looks up the contacts for each user
// in the idList and returns their contact object. It returns a buffered channel
// of contact objects which can be used to reset sessions. If a user cannot be
// found in user discovery, it returns a contact with an empty DhPubKey.
func LookupContacts(idList []*id.ID, udManager *bindings.UserDiscovery) chan *contact.Contact {
	contactsCh := make(chan *contact.Contact, len(idList))
	timeout := int(time.Duration(90 * time.Second))

	lookup := lookupcb{contactsCh: contactsCh}

	// TODO:
	//  numConcurrent := 8
	// then only run 8 at a time...
	for _, uid := range idList {
		go func() {
			udManager.Lookup(uid[:], lookup, timeout)
		}()
	}

	return contactsCh
}
