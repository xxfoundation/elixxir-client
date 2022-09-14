////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
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

// IsRegisteredWithUD is a function which checks the internal state
// files to determine if a user has registered with UD in the past.
//
// Parameters:
//  - e2eID -  REQUIRED. The tracked e2e object ID. This can be retrieved using [E2e.GetID].
//
// Returns:
//   - bool - A boolean representing true if the user has been registered with UD already
//            or false if it has not been registered already.
//  - error - An error should only be returned if the internal tracker failed to retrieve an
//            E2e object given the e2eId. If an error was returned, the registration state check
//            was not performed properly, and the boolean returned should be ignored.
func IsRegisteredWithUD(e2eId int) (bool, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return false, err
	}

	return ud.IsRegistered(user.api.GetStorage().GetKV()), nil
}

// NewOrLoadUd loads an existing UserDiscovery from storage or creates a new
// UserDiscovery if there is no storage data. Regardless of storage state,
// the UserDiscovery object returned will be registered with the
// User Discovery service. If the user is not already registered, a call
// to register will occur internally. If the user is already registered,
// this call will simply load state and return to you a UserDiscovery object.
// Some parameters are required for registering with the service, but are not required
// if the user is already registered. These will be noted in the parameters section as
// "SEMI-REQUIRED".
//
// Certain parameters are required every call to this function. These parameters are listed below
// as "REQUIRED". For example, parameters need be provided to specify how to connect to the
// User Discovery service. These parameters specifically may be used to contact either the UD
// server hosted by the xx network team or a custom third-party operated server. For the former,
// all the information may be fetched from the NDF using the bindings. These fetch
// methods are detailed in the parameters section.
//
// Params
//  - e2eID -  REQUIRED. The tracked e2e object ID. This is returned by [E2e.GetID].
//  - follower - REQUIRED. Network follower function. This will check if the network
//    follower is running.
//  - username - SEMI-REQUIRED. The username the user wants to register with UD.
//    If the user is already registered, this field may be blank. If the user is not
//    already registered, these field must be populated with a username that meets the
//    requirements of the UD service. For example, in the xx network's UD service,
//    the username must not be registered by another user.
//  - registrationValidationSignature - SEMI-REQUIRED. A signature provided by the xx network
//    (i.e. the client registrar). If the user is not already registered, this field is required
//    in order to register with the xx network. This may be nil if the user is already registered
//    or connecting to a third-party UD service unassociated with the xx network.
//  - cert - REQUIRED. The TLS certificate for the UD server this call will connect with.
//    If this is nil, you may not contact the UD server hosted by the xx network.
//    Third-party services may vary.
//    You may use the UD server run by the xx network team by using [E2e.GetUdCertFromNdf].
//  - contactFile - REQUIRED. The data within a marshalled [contact.Contact]. This represents the
//    contact file of the server this call will connect with.
//    If this is nil, you may not contact the UD server hosted by the xx network.
//    Third-party services may vary.
//    You may use the UD server run by the xx network team by using [E2e.GetUdContactFromNdf].
//  - address - REQUIRED. The IP address of the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using [E2e.GetUdAddressFromNdf].
//    If this is nil, you may not contact the UD server hosted by the xx network.
//    Third-party services may vary.
//
// Returns
//  - A Manager object which is registered to the specified UD service.
func NewOrLoadUd(e2eID int, follower UdNetworkStatus, username string,
	registrationValidationSignature, cert, contactFile []byte, address string) (
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
//  - username - The username this user registered with initially. This should
//               not be nullable, and be JSON marshalled as retrieved from
//               UserDiscovery.GetFacts().
//  - emailFactJson - nullable JSON marshalled email [fact.Fact]
//  - phoneFactJson - nullable JSON marshalled phone [fact.Fact]
//  - cert - the TLS certificate for the UD server this call will connect with.
//    You may use the UD server run by the xx network team by using
//    E2e.GetUdCertFromNdf.
//  - contactFile - the data within a marshalled contact.Contact. This
//    represents the contact file of the server this call will connect with. You
//    may use the UD server run by the xx network team by using
//    E2e.GetUdContactFromNdf.
//  - address - the IP address of the UD server this call will connect with. You
//    may use the UD server run by the xx network team by using
//    E2e.GetUdAddressFromNdf.
func NewUdManagerFromBackup(e2eID int, follower UdNetworkStatus,
	usernameJson, emailFactJson, phoneFactJson,
	cert, contactFile []byte, address string) (*UserDiscovery, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	var email, phone, username fact.Fact

	// Parse email if non-nil
	if emailFactJson != nil {
		err = json.Unmarshal(emailFactJson, &email)
		if err != nil {
			return nil, err
		}
	}

	// Parse phone if non-nil
	if phoneFactJson != nil {
		err = json.Unmarshal(phoneFactJson, &phone)
		if err != nil {
			return nil, err
		}
	}

	// Parse username
	err = json.Unmarshal(usernameJson, &username)
	if err != nil {
		return nil, err
	}

	UdNetworkStatusFn := func() xxdk.Status {
		return xxdk.Status(follower.UdNetworkStatus())
	}

	u, err := ud.NewManagerFromBackup(
		user.api, user.api.GetComms(), UdNetworkStatusFn,
		username, email, phone,
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

// ConfirmFact confirms a fact first registered via SendRegisterFact. The
// confirmation ID comes from SendRegisterFact while the code will come over the
// associated communications system.
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
//  - lookupId - the marshalled bytes of the id.ID object for the user that
//    LookupUD will look up.
//  - singleRequestParams - the JSON marshalled bytes of single.RequestParams
//
// Returns:
//  - []byte - the JSON marshalled bytes of the SingleUseSendReport object,
//    which can be passed into Cmix.WaitForRoundResult to see if the send
//    succeeded.
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
		ReceptionID: eid.Source,
		RoundsList:  makeRoundsList(rids...),
		RoundURL:    getRoundURL(rids[0]),
	}

	return json.Marshal(sr)
}

////////////////////////////////////////////////////////////////////////////////
// User Discovery MultiLookup                                                      //
////////////////////////////////////////////////////////////////////////////////

// UdMultiLookupCallback contains the callback called by MultiLookupUD that returns the
// contacts which match the passed in IDs.
//
// Parameters:
//  - contactListJSON - the JSON marshalled bytes of []contact.Contact, or nil
//    if an error occurs.
//
//   JSON Example:
//   {
//  	"<xxc(2)F8dL9EC6gy+RMJuk3R+Au6eGExo02Wfio5cacjBcJRwDEgB7Ugdw/BAr6RkCABkWAFV1c2VybmFtZTA7c4LzV05sG+DMt+rFB0NIJg==xxc>",
//  	"<xxc(2)eMhAi/pYkW5jCmvKE5ZaTglQb+fTo1D8NxVitr5CCFADEgB7Ugdw/BAr6RoCABkWAFV1c2VybmFtZTE7fElAa7z3IcrYrrkwNjMS2w==xxc>",
//  	"<xxc(2)d7RJTu61Vy1lDThDMn8rYIiKSe1uXA/RCvvcIhq5Yg4DEgB7Ugdw/BAr6RsCABkWAFV1c2VybmFtZTI7N3XWrxIUpR29atpFMkcR6A==xxc>"
//	}
//  - failedIDs - JSON marshalled list of []*id.ID objects which failed lookup
//  - err - any errors that occurred in the multilookup.
type UdMultiLookupCallback interface {
	Callback(contactListJSON []byte, failedIDs []byte, err error)
}

type lookupResp struct {
	id      *id.ID
	contact contact.Contact
	err     error
}

// MultiLookupUD returns the public key of all passed in IDs as known by the
// user discovery system or returns by the timeout.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - udContact - the marshalled bytes of the contact.Contact object
//  - lookupIds - JSON marshalled list of []*id.ID object for the users that
//    MultiLookupUD will look up.
//  - singleRequestParams - the JSON marshalled bytes of single.RequestParams
//
// Returns:
//  - []byte - the JSON marshalled bytes of the SingleUseSendReport object,
//    which can be passed into Cmix.WaitForRoundResult to see if the send
//    succeeded.
func MultiLookupUD(e2eID int, udContact []byte, cb UdMultiLookupCallback,
	lookupIds []byte, singleRequestParamsJSON []byte) error {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return err
	}

	c, err := contact.Unmarshal(udContact)
	if err != nil {
		return err
	}

	var idList []*id.ID
	err = json.Unmarshal(lookupIds, &idList)
	if err != nil {
		return err
	}

	var p single.RequestParams
	err = json.Unmarshal(singleRequestParamsJSON, &p)
	if err != nil {
		return err
	}

	respCh := make(chan lookupResp, len(idList))
	for _, uid := range idList {
		localID := uid.DeepCopy()
		callback := func(c contact.Contact, err error) {
			respCh <- lookupResp{
				id:      localID,
				contact: c,
				err:     err,
			}
		}
		go func() {
			_, _, err := ud.Lookup(user.api, c, callback, localID, p)
			if err != nil {
				respCh <- lookupResp{
					id:      localID,
					contact: contact.Contact{},
					err:     err,
				}
			}
		}()

	}

	go func() {
		var contactList []contact.Contact
		var failedIDs []*id.ID
		var errorString string
		for numReturned := 0; numReturned < len(idList); numReturned++ {
			response := <-respCh
			if response.err != nil {
				failedIDs = append(failedIDs, response.id)
				contactList = append(contactList, response.contact)
			} else {
				errorString = errorString +
					fmt.Sprintf("Failed to lookup id %s: %+v",
						response.id, response.err)
			}
		}

		marshalledFailedIds, err := json.Marshal(failedIDs)
		if err != nil {
			cb.Callback(nil, nil,
				errors.WithMessage(err,
					"Failed to marshal failed IDs"))
		}

		marshaledContactList := make([][]byte, 0)
		for _, con := range contactList {
			marshaledContactList = append(
				marshaledContactList, con.Marshal())
		}

		contactListJSON, err2 := json.Marshal(marshaledContactList)
		if err2 != nil {
			jww.FATAL.Panicf(
				"Failed to marshal list of contact.Contact: %+v", err2)
		}
		cb.Callback(contactListJSON, marshalledFailedIds, errors.New(errorString))
	}()

	return nil
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
//    if an error occurs.
//
//   JSON Example:
//   {
//  	"<xxc(2)F8dL9EC6gy+RMJuk3R+Au6eGExo02Wfio5cacjBcJRwDEgB7Ugdw/BAr6RkCABkWAFV1c2VybmFtZTA7c4LzV05sG+DMt+rFB0NIJg==xxc>",
//  	"<xxc(2)eMhAi/pYkW5jCmvKE5ZaTglQb+fTo1D8NxVitr5CCFADEgB7Ugdw/BAr6RoCABkWAFV1c2VybmFtZTE7fElAa7z3IcrYrrkwNjMS2w==xxc>",
//  	"<xxc(2)d7RJTu61Vy1lDThDMn8rYIiKSe1uXA/RCvvcIhq5Yg4DEgB7Ugdw/BAr6RsCABkWAFV1c2VybmFtZTI7N3XWrxIUpR29atpFMkcR6A==xxc>"
//	}
//  - err - any errors that occurred in the search.
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
//    which can be passed into Cmix.WaitForRoundResult to see if the send
//    operation succeeded.
func SearchUD(e2eID int, udContact []byte, cb UdSearchCallback,
	factListJSON, singleRequestParamsJSON []byte) ([]byte, error) {

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
		marshaledContactList := make([][]byte, 0)
		// fixme: it may be wiser to change this callback interface
		//   to simply do the work below when parsing the response from UD.
		//   that would change ud/search.go in two places:
		//    - searchCallback
		//    - parseContacts
		//  I avoid doing that as it changes interfaces w/o approval
		for i := range contactList {
			con := contactList[i]
			marshaledContactList = append(
				marshaledContactList, con.Marshal())
		}

		contactListJSON, err2 := json.Marshal(marshaledContactList)
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
		ReceptionID: eid.Source,
		RoundsList:  makeRoundsList(rids...),
		RoundURL:    getRoundURL(rids[0]),
	}

	return json.Marshal(sr)
}
