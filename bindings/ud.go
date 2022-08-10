////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

////////////////////////////////////////////////////////////////////////////////
// Singleton Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// udTrackerSingleton is used to track UserDiscovery objects so that they can be
// referenced by ID back over the bindings.
var udTrackerSingleton = &udTracker{
	tracked: make(map[int]*UserDiscovery),
	count:   0,
}

// udTracker is a singleton used to keep track of extant UserDiscovery objects,
// preventing race conditions created by passing it over the bindings.
type udTracker struct {
	tracked map[int]*UserDiscovery
	count   int
	mux     sync.RWMutex
}

// make create a UserDiscovery from an ud.Manager, assigns it a unique ID, and
// adds it to the udTracker.
func (ut *udTracker) make(u *ud.Manager) *UserDiscovery {
	ut.mux.Lock()
	defer ut.mux.Unlock()

	id := ut.count
	ut.count++

	ut.tracked[id] = &UserDiscovery{
		api: u,
		id:  id,
	}

	return ut.tracked[id]
}

// get an UserDiscovery from the udTracker given its ID.
func (ut *udTracker) get(id int) (*UserDiscovery, error) {
	ut.mux.RLock()
	defer ut.mux.RUnlock()

	c, exist := ut.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get UserDiscovery for ID %d, does not exist", id)
	}

	return c, nil
}

// delete removes a UserDiscovery from the udTracker.
func (ut *udTracker) delete(id int) {
	ut.mux.Lock()
	defer ut.mux.Unlock()

	delete(ut.tracked, id)
}

////////////////////////////////////////////////////////////////////////////////
// Structs and Interfaces                                                     //
////////////////////////////////////////////////////////////////////////////////

// UserDiscovery is a bindings-layer struct that wraps an ud.Manager interface.
type UserDiscovery struct {
	api *ud.Manager
	id  int
}

// GetID returns the udTracker ID for the UserDiscovery object.
func (ud *UserDiscovery) GetID() int {
	return ud.id
}

// UdNetworkStatus contains the UdNetworkStatus, which is a bindings-level
// interface for ud.udNetworkStatus.
type UdNetworkStatus interface {
	// UdNetworkStatus returns:
	// - int - a xxdk.Status int
	UdNetworkStatus() int
}

////////////////////////////////////////////////////////////////////////////////
// Manager functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// NewOrLoadUd loads an existing Manager from storage or creates a
// new one if there is no extant storage information. Parameters need be provided
// to specify how to connect to the User Discovery service. These parameters may be used
// to contact either the UD server hosted by the xx network team or a custom
// third-party operated server. For the former, all the information may be pulled from the
// NDF using the bindings.
//
// Params
//  - e2eID - e2e object ID in the tracker
//  - follower - network follower func wrapped in UdNetworkStatus
//  - username - the username the user wants to register with UD.
//    If the user is already registered, this field may be blank
//  - networkValidationSig is a signature provided by the network (i.e. the client registrar).
//    This may be nil, however UD may return an error in some cases (e.g. in a production level
//    environment).
//  - cert is the TLS certificate for the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdCertFromNdf.
//  - contactFile is the data within a marshalled contact.Contact. This represents the
//    contact file of the server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdContactFromNdf.
//  - address is the IP address of the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdAddressFromNdf.
//
// Returns
//  - A Manager object which is registered to the specified UD service.
func NewOrLoadUd(e2eID int, follower UdNetworkStatus,
	username string, registrationValidationSignature,
	cert, contactFile []byte, address string) (
	*UserDiscovery, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Construct callback
	UdNetworkStatusFn := func() xxdk.Status {
		return xxdk.Status(follower.UdNetworkStatus())
	}

	// Build manager
	u, err := ud.NewOrLoad(user.api, user.api.GetComms(),
		UdNetworkStatusFn, username, registrationValidationSignature,
		cert, contactFile, address)
	if err != nil {
		return nil, err
	}

	// Track and return manager
	return udTrackerSingleton.make(u), nil
}

// NewUdManagerFromBackup builds a new user discover manager from a backup. It
// will construct a manager that is already registered and restore already
// registered facts into store.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - follower - network follower func wrapped in UdNetworkStatus
//  - emailFactJson - nullable JSON marshalled email [fact.Fact]
//  - phoneFactJson - nullable JSON marshalled phone [fact.Fact]
//  - cert is the TLS certificate for the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdCertFromNdf.
//  - contactFile is the data within a marshalled contact.Contact. This represents the
//    contact file of the server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdContactFromNdf.
//  - address is the IP address of the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using E2e.GetUdAddressFromNdf.
func NewUdManagerFromBackup(e2eID int, follower UdNetworkStatus, emailFactJson,
	phoneFactJson []byte, cert, contactFile []byte, address string) (*UserDiscovery, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	var email, phone fact.Fact
	if emailFactJson != nil {
		err = json.Unmarshal(emailFactJson, &email)
		if err != nil {
			return nil, err
		}
	}

	if phoneFactJson != nil {
		err = json.Unmarshal(phoneFactJson, &phone)
		if err != nil {
			return nil, err
		}
	}

	UdNetworkStatusFn := func() xxdk.Status {
		return xxdk.Status(follower.UdNetworkStatus())
	}

	u, err := ud.NewManagerFromBackup(
		user.api, user.api.GetComms(), UdNetworkStatusFn,
		email, phone,
		cert, contactFile, address)
	if err != nil {
		return nil, err
	}

	return udTrackerSingleton.make(u), nil
}

// GetFacts returns a JSON marshalled list of [fact.Fact] objects that exist
// within the Store's registeredFacts map.
func (ud *UserDiscovery) GetFacts() []byte {
	jsonData, err := json.Marshal(ud.api.GetFacts())
	if err != nil {
		jww.FATAL.Panicf("Failed to JSON marshal fact list: %+v", err)
	}
	return jsonData
}

// GetContact returns the marshalled bytes of the contact.Contact for UD as
// retrieved from the NDF.
func (ud *UserDiscovery) GetContact() ([]byte, error) {
	return ud.api.GetContact().Marshal(), nil
}

// ConfirmFact confirms a fact first registered via AddFact. The confirmation ID
// comes from AddFact while the code will come over the associated
// communications system.
func (ud *UserDiscovery) ConfirmFact(confirmationID, code string) error {
	return ud.api.ConfirmFact(confirmationID, code)
}

// SendRegisterFact adds a fact for the user to user discovery. Will only
// succeed if the user is already registered and the system does not have the
// fact currently registered for any user.
//
// This does not complete the fact registration process, it returns a
// confirmation ID instead. Over the communications system the fact is
// associated with, a code will be sent. This confirmation ID needs to be called
// along with the code to finalize the fact.
//
// Parameters:
//  - factJson - a JSON marshalled [fact.Fact]
func (ud *UserDiscovery) SendRegisterFact(factJson []byte) (string, error) {
	var f fact.Fact
	err := json.Unmarshal(factJson, &f)
	if err != nil {
		return "", err
	}

	return ud.api.SendRegisterFact(f)
}

// PermanentDeleteAccount removes the username associated with this user from
// the UD service. This will only take a username type fact, and the fact must
// be associated with this user.
//
// Parameters:
//  - factJson - a JSON marshalled [fact.Fact]
func (ud *UserDiscovery) PermanentDeleteAccount(factJson []byte) error {
	var f fact.Fact
	err := json.Unmarshal(factJson, &f)
	if err != nil {
		return err
	}

	return ud.api.PermanentDeleteAccount(f)
}

// RemoveFact removes a previously confirmed fact. This will fail if the fact
// passed in is not UD service does not associate this fact with this user.
//
// Parameters:
//  - factJson - a JSON marshalled [fact.Fact]
func (ud *UserDiscovery) RemoveFact(factJson []byte) error {
	var f fact.Fact
	err := json.Unmarshal(factJson, &f)
	if err != nil {
		return err
	}

	return ud.api.RemoveFact(f)
}

////////////////////////////////////////////////////////////////////////////////
// User Discovery Lookup                                                      //
////////////////////////////////////////////////////////////////////////////////

// UdLookupCallback contains the callback called by LookupUD that returns the
// contact that matches the passed in ID.
//
// Parameters:
//  - contactBytes - the marshalled bytes of contact.Contact returned from the
//    lookup, or nil if an error occurs
//  - err - any errors that occurred in the lookup
type UdLookupCallback interface {
	Callback(contactBytes []byte, err error)
}

// LookupUD returns the public key of the passed ID as known by the user
// discovery system or returns by the timeout.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - udContact - the marshalled bytes of the contact.Contact object
//  - lookupId - the marshalled bytes of the id.ID object for the user
//    that LookupUD will look up.
//  - singleRequestParams - the JSON marshalled bytes of single.RequestParams
//
// Returns:
//  - []byte - the JSON marshalled bytes of the SingleUseSendReport object,
//    which can be passed into WaitForRoundResult to see if the send succeeded.
func LookupUD(e2eID int, udContact []byte, cb UdLookupCallback,
	lookupId []byte, singleRequestParamsJSON []byte) ([]byte, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	c, err := contact.Unmarshal(udContact)
	if err != nil {
		return nil, err
	}

	uid, err := id.Unmarshal(lookupId)
	if err != nil {
		return nil, err
	}

	var p single.RequestParams
	err = json.Unmarshal(singleRequestParamsJSON, &p)
	if err != nil {
		return nil, err
	}

	callback := func(c contact.Contact, err error) {
		cb.Callback(c.Marshal(), err)
	}

	rids, eid, err := ud.Lookup(user.api, c, callback, uid, p)
	if err != nil {
		return nil, err
	}

	sr := SingleUseSendReport{
		EphID:       eid.EphId.Int64(),
		ReceptionID: eid.Source.Marshal(),
		RoundsList:  makeRoundsList(rids...),
	}

	return json.Marshal(sr)
}

////////////////////////////////////////////////////////////////////////////////
// User Discovery Search                                                      //
////////////////////////////////////////////////////////////////////////////////

// UdSearchCallback contains the callback called by SearchUD that returns a list
// of contact.Contact objects  that match the list of facts passed into
// SearchUD.
//
// Parameters:
//  - contactListJSON - the JSON marshalled bytes of []contact.Contact, or nil
//    if an error occurs
//  - err - any errors that occurred in the search
type UdSearchCallback interface {
	Callback(contactListJSON []byte, err error)
}

// SearchUD searches user discovery for the passed Facts. The searchCallback
// will return a list of contacts, each having the facts it hit against. This is
// NOT intended to be used to search for multiple users at once; that can have a
// privacy reduction. Instead, it is intended to be used to search for a user
// where multiple pieces of information is known.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - udContact - the marshalled bytes of the contact.Contact for the user
//    discovery server
//  - factListJSON - the JSON marshalled bytes of [fact.FactList]
//  - singleRequestParams - the JSON marshalled bytes of single.RequestParams
//
// Returns:
//  - []byte - the JSON marshalled bytes of the SingleUseSendReport object,
//    which can be passed into WaitForRoundResult to see if the send succeeded.
func SearchUD(e2eID int, udContact []byte, cb UdSearchCallback,
	factListJSON []byte, singleRequestParamsJSON []byte) ([]byte, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	c, err := contact.Unmarshal(udContact)
	if err != nil {
		return nil, err
	}

	var list fact.FactList
	err = json.Unmarshal(factListJSON, &list)
	if err != nil {
		return nil, err
	}

	var p single.RequestParams
	err = json.Unmarshal(singleRequestParamsJSON, &p)
	if err != nil {
		return nil, err
	}

	callback := func(contactList []contact.Contact, err error) {
		contactListJSON, err2 := json.Marshal(contactList)
		if err2 != nil {
			jww.FATAL.Panicf(
				"Failed to marshal list of contact.Contact: %+v", err2)
		}

		cb.Callback(contactListJSON, err)
	}

	rids, eid, err := ud.Search(user.api, c, callback, list, p)
	if err != nil {
		return nil, err
	}

	sr := SingleUseSendReport{
		EphID:       eid.EphId.Int64(),
		ReceptionID: eid.Source.Marshal(),
		RoundsList:  makeRoundsList(rids...),
	}

	return json.Marshal(sr)
}
