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
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/primitives/fact"
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
// Main functions                                                             //
////////////////////////////////////////////////////////////////////////////////

// LoadOrNewUserDiscovery creates a bindings-level user discovery manager.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - follower - network follower func wrapped in UdNetworkStatus
func LoadOrNewUserDiscovery(e2eID int, follower UdNetworkStatus,
	username string, registrationValidationSignature []byte) (
	*UserDiscovery, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	UdNetworkStatusFn := func() xxdk.Status {
		return xxdk.Status(follower.UdNetworkStatus())
	}

	u, err := ud.LoadOrNewManager(user.api, user.api.GetComms(),
		UdNetworkStatusFn, username, registrationValidationSignature)
	if err != nil {
		return nil, err
	}

	return udTrackerSingleton.make(u), nil
}

// NewUdManagerFromBackup builds a new user discover manager from a backup. It
// will construct a manager that is already registered and restore already
// registered facts into store.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - follower - network follower func wrapped in UdNetworkStatus
//  - emailFactJson - a JSON marshalled email fact.Fact
//  - phoneFactJson - a JSON marshalled phone fact.Fact
func NewUdManagerFromBackup(e2eID int, follower UdNetworkStatus, emailFactJson,
	phoneFactJson []byte) (*UserDiscovery, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	var email, phone fact.Fact
	err = json.Unmarshal(emailFactJson, &email)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(phoneFactJson, &phone)
	if err != nil {
		return nil, err
	}

	UdNetworkStatusFn := func() xxdk.Status {
		return xxdk.Status(follower.UdNetworkStatus())
	}

	u, err := ud.NewManagerFromBackup(
		user.api, user.api.GetComms(), UdNetworkStatusFn, email, phone)
	if err != nil {
		return nil, err
	}

	return udTrackerSingleton.make(u), nil
}

// GetFacts returns a JSON marshalled list of fact.Fact objects that exist
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
	c, err := ud.api.GetContact()
	if err != nil {
		return nil, err
	}

	return json.Marshal(c)
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
//  - factJson - a JSON marshalled fact.Fact
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
//  - factJson - a JSON marshalled fact.Fact
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
//  - factJson - a JSON marshalled fact.Fact
func (ud *UserDiscovery) RemoveFact(factJson []byte) error {
	var f fact.Fact
	err := json.Unmarshal(factJson, &f)
	if err != nil {
		return err
	}

	return ud.api.RemoveFact(f)
}

// SetAlternativeUserDiscovery sets the alternativeUd object within manager.
// Once set, any user discovery operation will go through the alternative
// user discovery service.
//
// To undo this operation, use UnsetAlternativeUserDiscovery.
func (ud *UserDiscovery) SetAlternativeUserDiscovery(
	altCert, altAddress, contactFile []byte) error {
	return ud.api.SetAlternativeUserDiscovery(altCert, altAddress, contactFile)
}

// UnsetAlternativeUserDiscovery clears out the information from the Manager
// object.
func (ud *UserDiscovery) UnsetAlternativeUserDiscovery() error {
	return ud.api.UnsetAlternativeUserDiscovery()
}
