////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxmutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/xx_network/primitives/netTime"
	"math"
	"strings"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/ud"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

// RestoreContactsUpdater interface provides a callback function
// for receiving update information from RestoreContactsFromBackup.
type RestoreContactsUpdater interface {
	// RestoreContactsCallback is called to report the current # of contacts
	// that have been found and how many have been restored
	// against the total number that need to be
	// processed. If an error occurs it it set on the err variable as a
	// plain string.
	RestoreContactsCallback(numFound, numRestored, total int, err string)
}

// RestoreContactsFromBackup takes as input the jason output of the
// `NewClientFromBackup` function, unmarshals it into IDs, looks up
// each ID in user discovery, and initiates a session reset request.
// This function will not return until every id in the list has been sent a
// request. It should be called again and again until it completes.
// xxDK users should not use this function. This function is used by
// the mobile phone apps and are not intended to be part of the xxDK. It
// should be treated as internal functions specific to the phone apps.
func RestoreContactsFromBackup(backupPartnerIDs []byte, user *xxdk.E2e,
	udManager *ud.Manager,
	updatesCb RestoreContactsUpdater) ([]*id.ID, []*id.ID,
	[]error, error) {

	var restored, failed []*id.ID
	var errs []error

	// Constants/control settings
	numRoutines := 8
	maxChanSize := 10000
	restoreTimeout := time.Duration(30 * time.Second)

	update := func(numFound, numRestored, total int, err string) {
		if updatesCb != nil {
			updatesCb.RestoreContactsCallback(numFound, numRestored,
				total, err)
		}
	}

	store := stateStore{
		apiStore: user.GetStorage(),
	}

	// Unmarshal IDs and then check restore state
	var idList []*id.ID
	if err := json.Unmarshal(backupPartnerIDs, &idList); err != nil {
		return nil, nil, nil, err
	}
	lookupIDs, resetContacts, restored := checkRestoreState(idList, store)

	// State variables, how many we have looked up successfully
	// and how many we have already reset.
	totalCnt := len(idList)
	lookupCnt := len(resetContacts)
	resetCnt := totalCnt - len(resetContacts) - len(lookupIDs)

	// Before we start, report initial state
	update(lookupCnt, resetCnt, totalCnt, "")

	// Initialize channels
	chanSize := int(math.Min(float64(maxChanSize), float64(len(idList))))
	// Jobs are processed via the following pipeline:
	//   lookupCh -> foundCh -> resetContactCh -> restoredCh
	// foundCh and restoredCh are used to track progress
	lookupCh := make(chan *id.ID, chanSize)
	foundCh := make(chan *contact.Contact, chanSize)
	resetContactCh := make(chan *contact.Contact, chanSize)
	restoredCh := make(chan *contact.Contact, chanSize)
	failCh := make(chan failure, chanSize)

	// Start routines for processing
	lcWg := &sync.WaitGroup{}
	lcWg.Add(numRoutines)
	rsWg := &sync.WaitGroup{}
	rsWg.Add(numRoutines)
	for i := 0; i < numRoutines; i++ {
		go LookupContacts(lookupCh, foundCh, failCh, user, udManager.GetContact(), lcWg)
		go ResetSessions(resetContactCh, restoredCh, failCh, user,
			rsWg)
	}

	// Load channels based on previous state
	go func() {
		for i := range lookupIDs {
			lookupCh <- lookupIDs[i]
		}
	}()
	go func() {
		for i := range resetContacts {
			lookupCnt += 1
			resetContactCh <- resetContacts[i]
		}
	}()

	// Failure processing, done separately (in a single thread)
	// because failures should not reset the timer
	failWg := sync.WaitGroup{}
	failWg.Add(1)
	go func() {
		defer failWg.Done()
		for fail := range failCh {
			failed = append(failed, fail.ID)
			errs = append(errs, fail.Err)
		}
	}()

	// Event Processing
	done := false
	var err error
	for !done {
		// NOTE: Timer is reset every loop
		timeoutTimer := time.NewTimer(restoreTimeout)
		select {
		case <-timeoutTimer.C:
			err = errors.New("restoring accounts timed out")
			done = true
		case c := <-foundCh:
			store.set(c, contactFound)
			lookupCnt += 1
			// NOTE: Prevent blocking by using routine here
			go func() { resetContactCh <- c }()
		case c := <-restoredCh:
			store.set(c, contactRestored)
			restored = append(restored, c.ID)
			resetCnt += 1
		}
		if resetCnt == totalCnt {
			done = true
		}
		update(lookupCnt, resetCnt, totalCnt, "")
	}

	// Cleanup
	//   lookupCh -> foundCh -> resetContactCh -> restoredCh
	close(lookupCh)
	// Now wait for subroutines to close before closing their output chans
	lcWg.Wait()
	// Close input to reset chan after lookup is done to avoid writes after
	// close
	close(foundCh)
	close(resetContactCh)
	rsWg.Wait()
	// failCh is closed after exit of the threads to avoid writes after
	// close
	close(failCh)
	close(restoredCh)
	failWg.Wait()

	return restored, failed, errs, err
}

// LookupContacts routine looks up contacts
// xxDK users should not use this function. This function is used by
// the mobile phone apps and are not intended to be part of the xxDK. It
// should be treated as internal functions specific to the phone apps.
func LookupContacts(in chan *id.ID, out chan *contact.Contact,
	failCh chan failure, user *xxdk.E2e, udContact contact.Contact,
	wg *sync.WaitGroup) {
	defer wg.Done()
	// Start looking up contacts with user discovery and feed this
	// contacts channel.
	for lookupID := range in {
		c, err := LookupContact(lookupID, user, udContact)
		if err == nil {
			out <- c
			continue
		}
		// If an error, figure out if I should report or retry
		errStr := err.Error()
		if strings.Contains(errStr, "failed to lookup ID") {
			failCh <- failure{ID: lookupID, Err: err}
			continue
		}
		jww.WARN.Printf("could not lookup %s: %v", lookupID, err)
	}
}

// ResetSessions routine reads the in channel, sends a reset session
// request, then marks it done by sending to the out channel.
// xxDK users should not use this function. This function is used by
// the mobile phone apps and are not intended to be part of the xxDK. It
// should be treated as internal functions specific to the phone apps.
func ResetSessions(in, out chan *contact.Contact, failCh chan failure,
	user *xxdk.E2e, wg *sync.WaitGroup) {
	defer wg.Done()
	for c := range in {
		_, err := user.GetAuth().Reset(*c)
		if err == nil {
			out <- c
			continue
		}
		// If an error, figure out if I should report or retry
		// Note: Always fail here for now.
		jww.WARN.Printf("could not reset %s: %v", c.ID, err)
		failCh <- failure{ID: c.ID, Err: err}
	}
}

// LookupContact lookups up a contact using the user discovery manager
// xxDK users should not use this function. This function is used by
// the mobile phone apps and are not intended to be part of the xxDK. It
// should be treated as internal functions specific to the phone apps.
func LookupContact(userID *id.ID, user *xxdk.E2e, udContact contact.Contact) (
	*contact.Contact, error) {
	// This is a little wonky, but wait until we get called then
	// set the result to the contact objects details if there is
	// no error
	waiter := sync.Mutex{}
	var result *contact.Contact
	var err error
	lookupCB := func(c contact.Contact, myErr error) {
		defer waiter.Unlock()
		if myErr != nil {
			err = myErr
		}
		result = &c
	}
	// Take lock once to make sure I will wait
	waiter.Lock()

	// in MS, so 90 seconds
	_, _, err = ud.Lookup(user, udContact, lookupCB, userID,
		single.GetDefaultRequestParams())

	// Now force a wait for callback to exit
	waiter.Lock()
	defer waiter.Unlock()

	return result, err
}

// restoreState is the internal state of a contact
type restoreState byte

const (
	contactNotFound restoreState = iota
	contactFound
	contactRestored
)

type failure struct {
	ID  *id.ID
	Err error
}

// //
// stateStore wraps a kv and stores contact state for the restoration
// TODO: Right now, it uses 1 contact-per-key approach, but it might make sense
// to wrap this in a mutex and load/store a whole list
// //
const stateStoreFmt = "restoreContactsFromBackup/v1/%s"

type stateStore struct {
	apiStore storage.Session
	// TODO: We could put a syncmap or something here instead of
	// 1-key-per-id
}

func (s stateStore) key(id *id.ID) string {
	return fmt.Sprintf(stateStoreFmt, id)
}

func (s stateStore) set(user *contact.Contact, state restoreState) error {
	key := s.key(user.ID)
	// First byte is state var, second is contact object
	data := []byte{byte(state)}
	data = append(data, user.Marshal()...)
	val := &versioned.Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      data,
	}
	return s.apiStore.Set(key, val)
}
func (s stateStore) get(id *id.ID) (restoreState, *contact.Contact, error) {
	key := s.key(id)
	val, err := s.apiStore.Get(key)
	if err != nil {
		return contactNotFound, nil, err
	}
	user, err := contact.Unmarshal(val.Data[1:])
	if err != nil {
		return contactFound, nil, err
	}
	return restoreState(val.Data[0]), &user, nil
}

// stateStore END

func checkRestoreState(IDs []*id.ID, store stateStore) ([]*id.ID,
	[]*contact.Contact, []*id.ID) {
	var idsToLookup []*id.ID
	var contactsToReset []*contact.Contact
	var contactsRestored []*id.ID
	for i := range IDs {
		id := IDs[i]
		idState, user, err := store.get(id)
		if err != nil {
			// Ignore errors here since they always will result
			// in a retry.
			jww.WARN.Printf("Error on restore check for %s: %v",
				id, err)
		}
		switch idState {
		case contactNotFound:
			idsToLookup = append(idsToLookup, id)
		case contactFound:
			contactsToReset = append(contactsToReset, user)
		case contactRestored:
			contactsRestored = append(contactsRestored, user.ID)
		}
	}
	return idsToLookup, contactsToReset, contactsRestored
}
