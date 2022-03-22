///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// This package wraps the user discovery system

type UserDiscovery struct {
	ud *ud.Manager
}

// NewUserDiscovery returns a new user discovery object. Only call this once. It must be called
// after StartNetworkFollower is called and will fail if the network has never
// been contacted.
// This function technically has a memory leak because it causes both sides of
// the bindings to think the other is in charge of the client object.
// In general this is not an issue because the client object should exist
// for the life of the program.
// This must be called while start network follower is running.
func NewUserDiscovery(client *Client) (*UserDiscovery, error) {
	single, err := client.getSingle()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create User Discovery Manager")
	}
	m, err := ud.NewManager(&client.api, single)

	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create User Discovery Manager")
	} else {
		return &UserDiscovery{ud: m}, nil
	}
}

// Register registers a user with user discovery. Will return an error if the
// network signatures are malformed or if the username is taken. Usernames
// cannot be changed after registration at this time. Will fail if the user is
// already registered.
// Identity does not go over cmix, it occurs over normal communications
func (ud *UserDiscovery) Register(username string) error {
	return ud.ud.Register(username)
}

// AddFact adds a fact for the user to user discovery. Will only succeed if the
// user is already registered and the system does not have the fact currently
// registered for any user.
// Will fail if the fact string is not well formed.
// This does not complete the fact registration process, it returns a
// confirmation id instead. Over the communications system the fact is
// associated with, a code will be sent. This confirmation ID needs to be
// called along with the code to finalize the fact.
func (ud *UserDiscovery) AddFact(fStr string) (string, error) {
	f, err := fact.UnstringifyFact(fStr)
	if err != nil {
		return "", errors.WithMessage(err, "Failed to add due to "+
			"malformed fact")
	}

	return ud.ud.SendRegisterFact(f)
}

// ConfirmFact confirms a fact first registered via AddFact. The confirmation ID comes from
// AddFact while the code will come over the associated communications system
func (ud *UserDiscovery) ConfirmFact(confirmationID, code string) error {
	return ud.ud.SendConfirmFact(confirmationID, code)
}

// RemoveFact removes a previously confirmed fact.  Will fail if the passed fact string is
// not well-formed or if the fact is not associated with this client.
// Users cannot remove username facts and must instead remove the user.
func (ud *UserDiscovery) RemoveFact(fStr string) error {
	f, err := fact.UnstringifyFact(fStr)
	if err != nil {
		return errors.WithMessage(err, "Failed to remove due to "+
			"malformed fact")
	}
	return ud.ud.RemoveFact(f)
}

// RemoveUser deletes a user. The fact sent must be the username.
// This function preserves the username forever and makes it
// unusable.
func (ud *UserDiscovery) RemoveUser(fStr string) error {
	f, err := fact.UnstringifyFact(fStr)
	if err != nil {
		return errors.WithMessage(err, "Failed to remove due to "+
			"malformed fact")
	}
	return ud.ud.RemoveUser(f)
}

//BackUpMissingFacts adds a registered fact to the Store object and saves
// it to storage. It can take in both an email or a phone number, passed into
// the function in that order.  Any one of these fields may be empty,
// however both fields being empty will cause an error. Any other fact that is not
// an email or phone number will return an error. You may only add a fact for the
// accepted types once each. If you attempt to back up a fact type that has already
// been backed up, an error will be returned. Anytime an error is returned, it means
// the backup was not successful.
// NOTE: Do not use this as a direct store operation. This feature is intended to add facts
// to a backend store that have ALREADY BEEN REGISTERED on the account.
// THIS IS NOT FOR ADDING NEWLY REGISTERED FACTS. That is handled on the backend.
func (ud *UserDiscovery) BackUpMissingFacts(email, phone string) error {
	var emailFact, phoneFact fact.Fact
	var err error
	if len(email) > 2 {
		emailFact, err = fact.UnstringifyFact(email)
		if err != nil {
			return errors.WithMessagef(err, "Failed to parse malformed email fact: %s", email)
		}
	}

	if len(phone) > 2 {
		phoneFact, err = fact.UnstringifyFact(phone)
		if err != nil {
			return errors.WithMessagef(err, "Failed to parse malformed phone fact: %s", phone)
		}
	}

	return ud.ud.BackUpMissingFacts(emailFact, phoneFact)
}

// SearchCallback returns the result of a search
type SearchCallback interface {
	Callback(contacts *ContactList, error string)
}

// Search for the passed Facts.  The factList is the stringification of a
// fact list object, look at /bindings/list.go for more on that object.
// This will reject if that object is malformed. The SearchCallback will return
// a list of contacts, each having the facts it hit against.
// This is NOT intended to be used to search for multiple users at once, that
// can have a privacy reduction. Instead, it is intended to be used to search
// for a user where multiple pieces of information is known.
func (ud UserDiscovery) Search(fl string, callback SearchCallback,
	timeoutMS int) error {
	factList, _, err := fact.UnstringifyFactList(fl)
	if err != nil {
		return errors.WithMessage(err, "Failed to search due to "+
			"malformed fact list")
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond
	cb := func(cl []contact.Contact, err error) {
		var contactList *ContactList
		var errStr string
		if err == nil {
			contactList = &ContactList{list: cl}
		} else {
			errStr = err.Error()
		}
		callback.Callback(contactList, errStr)
	}
	return ud.ud.Search(factList, cb, timeout)
}

// SingleSearchCallback returns the result of a single search
type SingleSearchCallback interface {
	Callback(contact *Contact, error string)
}

// SearchSingle searches for the passed Facts.  The fact is the stringification of a
// fact object, look at /bindings/contact.go for more on that object.
// This will reject if that object is malformed. The SearchCallback will return
// a list of contacts, each having the facts it hit against.
// This only searches for a single fact at a time. It is intended to make some
// simple use cases of the API easier.
func (ud UserDiscovery) SearchSingle(f string, callback SingleSearchCallback,
	timeoutMS int) error {
	fObj, err := fact.UnstringifyFact(f)
	if err != nil {
		return errors.WithMessage(err, "Failed to single search due "+
			"to malformed fact")
	}
	timeout := time.Duration(timeoutMS) * time.Millisecond
	cb := func(cl []contact.Contact, err error) {
		var c *Contact
		var errStr string
		if err == nil {
			c = &Contact{c: &cl[0]}
		} else {
			errStr = err.Error()
		}
		callback.Callback(c, errStr)
	}
	return ud.ud.Search([]fact.Fact{fObj}, cb, timeout)
}

// LookupCallback returns the result of a single lookup
type LookupCallback interface {
	Callback(contact *Contact, error string)
}

// Lookup the contact object associated with the given userID.  The
// id is the byte representation of an id.
// This will reject if that id is malformed. The LookupCallback will return
// the associated contact if it exists.
func (ud UserDiscovery) Lookup(idBytes []byte, callback LookupCallback,
	timeoutMS int) error {

	uid, err := id.Unmarshal(idBytes)
	if err != nil {
		return errors.WithMessage(err, "Failed to lookup due to "+
			"malformed id")
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond
	cb := func(cl contact.Contact, err error) {
		var c *Contact
		var errStr string
		if err == nil {
			c = &Contact{c: &cl}
		} else {
			errStr = err.Error()
		}
		callback.Callback(c, errStr)
	}

	return ud.ud.Lookup(uid, cb, timeout)

}

// MultiLookupCallback returns the result of many parallel lookups
type MultiLookupCallback interface {
	Callback(Succeeded *ContactList, failed *IdList, errors string)
}

type lookupResponse struct {
	C     contact.Contact
	err   error
	index int
	id    *id.ID
}

// MultiLookup Looks for the contact object associated with all given userIDs.
// The ids are the byte representation of an id stored in an IDList object.
// This will reject if that id is malformed or if the indexing on the IDList
// object is wrong. The MultiLookupCallback will return with all contacts
// returned within the timeout.
func (ud UserDiscovery) MultiLookup(ids *IdList, callback MultiLookupCallback,
	timeoutMS int) error {

	idList := make([]*id.ID, 0, ids.Len())

	//extract all IDs from
	for i := 0; i < ids.Len(); i++ {
		idBytes, err := ids.Get(i)
		if err != nil {
			return errors.WithMessagef(err, "Failed to get ID at index %d", i)
		}
		uid, err := id.Unmarshal(idBytes)
		if err != nil {
			return errors.WithMessagef(err, "Failed to lookup due to "+
				"malformed id at index %d", i)
		}
		idList = append(idList, uid)
	}

	//make the channels for the requests
	results := make(chan lookupResponse, len(idList))

	timeout := time.Duration(timeoutMS) * time.Millisecond

	//loop through the IDs and send the lookup
	for i := range idList {
		locali := i
		localID := idList[locali]
		cb := func(c contact.Contact, err error) {
			results <- lookupResponse{
				C:     c,
				err:   err,
				index: locali,
				id:    localID,
			}
		}

		go func() {
			err := ud.ud.Lookup(localID, cb, timeout)
			if err != nil {
				results <- lookupResponse{
					C: contact.Contact{},
					err: errors.WithMessagef(err, "Failed to send lookup "+
						"for user %s[%d]", localID, locali),
					index: locali,
					id:    localID,
				}
			}
		}()
	}

	//run the result gathering in its own thread
	go func() {
		returnedContactList := make([]contact.Contact, 0, len(idList))
		failedIDList := make([]*id.ID, 0, len(idList))
		var concatonatedErrs string

		//get the responses and return
		for numReturned := 0; numReturned < len(idList); numReturned++ {
			response := <-results
			if response.err == nil {
				returnedContactList = append(returnedContactList, response.C)
			} else {
				failedIDList = append(failedIDList, response.id)
				concatonatedErrs = concatonatedErrs + fmt.Sprintf("Error returned from "+
					"send to %d [%d]:%+v\t", response.id, response.index, response.err)
			}
		}

		callback.Callback(&ContactList{list: returnedContactList}, &IdList{list: failedIDList}, concatonatedErrs)
	}()

	return nil
}

// SetAlternativeUserDiscovery sets the alternativeUd object within manager.
// Once set, any user discovery operation will go through the alternative
// user discovery service.
// To undo this operation, use UnsetAlternativeUserDiscovery.
// The contact file is the already read in bytes, not the file path for the contact file.
func (ud *UserDiscovery) SetAlternativeUserDiscovery(address, cert, contactFile []byte) error {
	return ud.ud.SetAlternativeUserDiscovery(cert, address, contactFile)
}

// UnsetAlternativeUserDiscovery clears out the information from
// the Manager object.
func (ud *UserDiscovery) UnsetAlternativeUserDiscovery() error {
	return ud.ud.UnsetAlternativeUserDiscovery()
}

func WrapUserDiscovery(ud *ud.Manager) *UserDiscovery {
	return &UserDiscovery{ud: ud}
}
