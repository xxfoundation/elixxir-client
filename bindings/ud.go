///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

//This package wraps the user discovery system

// User Discovery object
type UserDiscovery struct{
	ud *ud.Manager
}

// Returns a new user discovery object. Only call this once. It must be called
// after StartNetworkFollower is called and will fail if the network has never
// been contacted.
// This function technically has a memory leak because it causes both sides of
// the bindings to think the other is in charge of the client object.
// In general this is not an issue because the client object should exist
// for the life of the program.
func NewUserDiscovery(client *Client)(*UserDiscovery, error){
	m, err := ud.NewManager(&client.api)

	if err!=nil{
		return nil, err
	}else{
		return &UserDiscovery{ud:m}, nil
	}
}

// Register registers a user with user discovery. Will return an error if the
// network signatures are malformed or if the username is taken. Usernames
// cannot be changed after registration at this time. Will fail if the user is
// already registered.
// Identity does not go over cmix, it occurs over normal communications
func (ud *UserDiscovery)Register(username string)error{
	return ud.ud.Register(username)
}

// Adds a fact for the user to user discovery. Will only succeed if the
// user is already registered and the system does not have the fact currently
// registered for any user.
// Will fail if the fact string is not well formed.
// This does not complete the fact registration process, it returns a
// confirmation id instead. Over the communications system the fact is
// associated with, a code will be sent. This confirmation ID needs to be
// called along with the code to finalize the fact.
func (ud *UserDiscovery)AddFact(fStr string)(string, error){
	f, err := fact.UnstringifyFact(fStr)
	if err !=nil{
		return "", errors.WithMessage(err, "Failed to add due to " +
			"malformed fact")
	}

	return ud.ud.SendRegisterFact(f)
}

// Confirms a fact first registered via AddFact. The confirmation ID comes from
// AddFact while the code will come over the associated communications system
func (ud *UserDiscovery)ConfirmFact(confirmationID, code string)error{
	return ud.ud.SendConfirmFact(confirmationID, code)
}

// Removes a previously confirmed fact.  Will fail if the passed fact string is
// not well formed or if the fact is not associated with this client.
func (ud *UserDiscovery)RemoveFact(fStr string)error{
	f, err := fact.UnstringifyFact(fStr)
	if err !=nil{
		return errors.WithMessage(err, "Failed to remove due to " +
			"malformed fact")
	}
	return ud.ud.RemoveFact(f)
}

// SearchCallback returns the result of a search
type SearchCallback interface {
	Callback(contacts *ContactList, error string)
}

// Searches for the passed Facts.  The factList is the stringification of a
// fact list object, look at /bindings/list.go for more on that object.
// This will reject if that object is malformed. The SearchCallback will return
// a list of contacts, each having the facts it hit against.
// This is NOT intended to be used to search for multiple users at once, that
// can have a privacy reduction. Instead, it is intended to be used to search
// for a user where multiple pieces of information is known.
func (ud UserDiscovery)Search(fl string, callback SearchCallback,
	timeoutMS int)error{
	factList, _, err := fact.UnstringifyFactList(fl)
	if err!=nil{
		return errors.WithMessage(err, "Failed to search due to " +
			"malformed fact list")
	}
	timeout := time.Duration(timeoutMS)*time.Millisecond
	cb := func(cl []contact.Contact, err error){
		var contactList *ContactList
		var errStr string
		if err==nil{
			contactList = &ContactList{list:cl}
		}else{
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

// Searches for the passed Facts.  The fact is the stringification of a
// fact object, look at /bindings/contact.go for more on that object.
// This will reject if that object is malformed. The SearchCallback will return
// a list of contacts, each having the facts it hit against.
// This only searches for a single fact at a time. It is intended to make some
// simple use cases of the API easier.
func (ud UserDiscovery)SearchSingle(f string, callback SingleSearchCallback,
	timeoutMS int)error{
	fObj, err := fact.UnstringifyFact(f)
	if err!=nil{
		return errors.WithMessage(err, "Failed to single search due " +
			"to malformed fact")
	}
	timeout := time.Duration(timeoutMS)*time.Millisecond
	cb := func(cl []contact.Contact, err error){
		var contact *Contact
		var errStr string
		if err==nil{
			contact = &Contact{c:&cl[0]}
		}else{
			errStr = err.Error()
		}
		callback.Callback(contact, errStr)
	}
	return ud.ud.Search([]fact.Fact{fObj}, cb, timeout)
}

// SingleSearchCallback returns the result of a single search
type LookupCallback interface {
	Callback(contact *Contact, error string)
}

// Looks for the contact object associated with the given userID.  The
// id is the byte representation of an id.
// This will reject if that id is malformed. The LookupCallback will return
// the associated contact if it exists.
func (ud UserDiscovery)Lookup(idBytes []byte, callback LookupCallback,
	timeoutMS int)error {

	uid, err := id.Unmarshal(idBytes)
	if err!=nil{
		return errors.WithMessage(err, "Failed to lookup due to " +
			"malformed id")
	}

	timeout := time.Duration(timeoutMS)*time.Millisecond
	cb := func(cl contact.Contact, err error){
		var contact *Contact
		var errStr string
		if err==nil{
			contact = &Contact{c:&cl}
		}else{
			errStr = err.Error()
		}
		callback.Callback(contact, errStr)
	}

	return ud.ud.Lookup(uid, cb, timeout)

}