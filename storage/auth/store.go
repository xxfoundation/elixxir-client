///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const NoRequest = "Request Not Found"

const storePrefix = "requestMap"
const requestMapKey = "map"

const requestMapVersion = 0

type Store struct {
	kv           *versioned.KV
	grp          *cyclic.Group
	receivedByID map[authIdentity]*ReceivedRequest
	sentByID     map[authIdentity]*SentRequest

	previousNegotiations map[id.ID]struct{}

	defaultID *id.ID

	srh SentRequestHandler

	mux sync.RWMutex
}

// NewStore creates a new store. All passed in private keys are added as
// sentByFingerprints so they can be used to trigger receivedByID.
func NewStore(kv *versioned.KV, grp *cyclic.Group, srh SentRequestHandler) error {
	kv = kv.Prefix(storePrefix)
	s := &Store{
		kv:                   kv,
		grp:                  grp,
		receivedByID:         make(map[authIdentity]*ReceivedRequest),
		sentByID:             make(map[authIdentity]*SentRequest),
		previousNegotiations: make(map[id.ID]struct{}),
		srh:                  srh,
	}

	err := s.savePreviousNegotiations()
	if err != nil {
		return errors.Errorf(
			"failed to load previousNegotiations partners: %+v", err)
	}

	return s.save()
}

// LoadStore loads an extant new store. All passed in private keys are added as
// sentByFingerprints so they can be used to trigger receivedByID.
func LoadStore(kv *versioned.KV, defaultID *id.ID, grp *cyclic.Group, srh SentRequestHandler) (*Store, error) {
	kv = kv.Prefix(storePrefix)

	s := &Store{
		kv:                   kv,
		grp:                  grp,
		receivedByID:         make(map[authIdentity]*ReceivedRequest),
		sentByID:             make(map[authIdentity]*SentRequest),
		previousNegotiations: make(map[id.ID]struct{}),
		defaultID:            defaultID,
		srh:                  srh,
	}

	var requestList []requestDisk

	//load all receivedByID
	sentObj, err := kv.Get(requestMapKey, requestMapVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load requestMap")
	}

	if err := json.Unmarshal(sentObj.Data, &requestList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to "+
			"unmarshal SentRequestMap")
	}

	jww.TRACE.Printf("%d found when loading AuthStore", len(requestList))

	//process receivedByID
	for _, rDisk := range requestList {

		requestType := RequestType(rDisk.T)

		partner, err := id.Unmarshal(rDisk.ID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load stored id: %+v", err)
		}

		var myID *id.ID
		// load the self id used. If it is blank, that means this is an older
		// version request, and replace it with the default ID
		if len(rDisk.MyID) == 0 {
			myID = defaultID
		} else {
			myID, err = id.Unmarshal(rDisk.ID)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored self id: %+v", err)
			}
		}

		switch requestType {
		case Sent:
			sr, err := loadSentRequest(kv, partner, myID, grp)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored sentRequest: %+v", err)
			}

			s.sentByID[sr.getAuthID()] = sr
			s.srh.Add(sr)
		case Receive:
			rr, err := loadReceivedRequest(kv, partner, myID)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored receivedRequest: %+v", err)
			}

			s.receivedByID[rr.aid] = rr

		default:
			jww.FATAL.Panicf("Unknown request type: %d", requestType)
		}
	}

	// Load previous negotiations from storage
	s.previousNegotiations, err = s.newOrLoadPreviousNegotiations()
	if err != nil {
		return nil, errors.Errorf("failed to load list of previouse "+
			"negotation partner IDs: %+v", err)
	}

	return s, nil
}

func (s *Store) save() error {
	requestIDList := make([]requestDisk, 0, len(s.receivedByID)+len(s.sentByID))
	for _, r := range s.receivedByID {
		rDisk := requestDisk{
			T:    uint(r.getType()),
			ID:   r.partner.ID.Marshal(),
			MyID: r.myID.Marshal(),
		}
		requestIDList = append(requestIDList, rDisk)
	}

	for _, r := range s.sentByID {
		rDisk := requestDisk{
			T:    uint(r.getType()),
			ID:   r.partner.Marshal(),
			MyID: r.myID.Marshal(),
		}
		requestIDList = append(requestIDList, rDisk)
	}

	data, err := json.Marshal(&requestIDList)
	if err != nil {
		return err
	}
	obj := versioned.Object{
		Version:   requestMapVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return s.kv.Set(requestMapKey, requestMapVersion, &obj)
}

func (s *Store) AddSent(partner, myID *id.ID, partnerHistoricalPubKey, myPrivKey,
	myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey, sidHPubA *sidh.PublicKey,
	fp format.Fingerprint) (*SentRequest, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	aid := makeAuthIdentity(partner, myID)

	if _, ok := s.sentByID[aid]; ok {
		return nil, errors.Errorf("Cannot make new sentRequest for partner "+
			"%s, one already exists", partner)
	}

	sr, err := newSentRequest(s.kv, partner, myID, partnerHistoricalPubKey, myPrivKey,
		myPubKey, sidHPrivA, sidHPubA, fp)

	if err != nil {
		return nil, err
	}

	s.sentByID[sr.getAuthID()] = sr
	s.srh.Add(sr)

	return sr, nil
}

func (s *Store) AddReceived(myID *id.ID, c contact.Contact, key *sidh.PublicKey) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("AddReceived new contact: %s with %s", c.ID, myID)

	aih := makeAuthIdentity(c.ID, myID)

	if _, ok := s.receivedByID[aih]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}

	r := newReceivedRequest(s.kv, myID, c, key)

	s.receivedByID[r.aid] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partner %s", c.ID)
	}

	return nil
}

// GetReceivedRequest returns the contact representing the partner request, if
// it exists. If it returns, then it takes the lock to ensure that there is only
// one operator at a time. The user of the API must release the lock by calling
// store.delete() or store.Failed() with the partner ID.
func (s *Store) GetReceivedRequest(partner, myID *id.ID) (*ReceivedRequest, error) {
	aid := makeAuthIdentity(partner, myID)

	s.mux.RLock()
	r, ok := s.receivedByID[aid]
	s.mux.RUnlock()

	if !ok {
		return nil, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	// Take the lock to ensure there is only one operator at a time
	r.mux.Lock()

	// Check that the request still exists; it could have been deleted while the
	// lock was taken
	s.mux.RLock()
	_, ok = s.receivedByID[aid]
	s.mux.RUnlock()

	if !ok {
		r.mux.Unlock()
		return nil, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return r, nil
}

// GetReceivedRequestData returns the contact representing the partner request
// if it exists. It does not take the lock. It is only meant to return the
// contact to an external API user.
func (s *Store) GetReceivedRequestData(partner, myID *id.ID) (contact.Contact, error) {
	if myID == nil {
		myID = s.defaultID
	}

	aid := makeAuthIdentity(partner, myID)

	s.mux.RLock()
	r, ok := s.receivedByID[aid]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return r.partner, nil
}

// Done is one of two calls after using a request. This one is to be used when
// the use is unsuccessful. It will allow any thread waiting on access to
// continue using the structure.
// It does not return an error because an error is not handleable.
func (s *Store) Done(rr *ReceivedRequest) {
	s.mux.RLock()
	r, ok := s.receivedByID[rr.aid]
	s.mux.RUnlock()

	r.mux.Unlock()

	if !ok {
		jww.ERROR.Panicf("Request cannot be finished, not "+
			"found: %s, %s", rr.partner, rr.myID)
		return
	}
}

// Delete is one of two calls after using a request. This one is to be used when
// the use is unsuccessful. It deletes all references to the request associated
// with the passed partner, if it exists. It will allow any thread waiting on
// access to continue. They should fail due to the deletion of the structure.
func (s *Store) Delete(partner *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	r, ok := s.receivedByID[*partner]

	if !ok {
		return errors.Errorf("Request not found: %s", partner)
	}

	switch r.rt {
	case Sent:
		s.deleteSentRequest(r)
	case Receive:
		s.deleteReceiveRequest(r)
	}

	delete(s.receivedByID, *partner)
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deletion: %+v", err)
	}

	err := s.deletePreviousNegotiationPartner(partner)
	if err != nil {
		jww.FATAL.Panicf("Failed to delete partner negotiations: %+v", err)
	}

	return nil
}

// DeleteAllRequests clears the request map and all associated storage objects
// containing request data.
func (s *Store) DeleteAllRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.receivedByID, partnerId)
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.receivedByID, partnerId)
		}

	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all receivedByID: %+v", err)
	}

	return nil
}

// DeleteRequest deletes a request from Store given a partner ID.
// If the partner ID exists as a request,  then the request will be deleted
// and the state stored. If the partner does not exist, then an error will
// be returned.
func (s *Store) DeleteRequest(partnerId *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	req, ok := s.receivedByID[*partnerId]
	if !ok {
		return errors.Errorf("Request for %s does not exist", partnerId)
	}

	switch req.rt {
	case Sent:
		s.deleteSentRequest(req)
	case Receive:
		s.deleteReceiveRequest(req)
	}

	delete(s.receivedByID, *partnerId)

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting partner request for partner %s: %+v", partnerId, err)
	}

	return nil
}

// DeleteSentRequests deletes all Sent receivedByID from Store.
func (s *Store) DeleteSentRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.receivedByID, partnerId)
		case Receive:
			continue
		}
	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all sent receivedByID: %+v", err)
	}

	return nil
}

// DeleteReceiveRequests deletes all Receive receivedByID from Store.
func (s *Store) DeleteReceiveRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.receivedByID {
		switch req.rt {
		case Sent:
			continue
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.receivedByID, partnerId)
		}
	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all partner receivedByID: %+v", err)
	}

	return nil
}

// deleteSentRequest is a helper function which deletes a Sent request from storage.
func (s *Store) deleteSentRequest(r *ReceivedRequest) {
	delete(s.sentByFingerprints, r.sent.fingerprint)
	if err := r.sent.delete(); err != nil {
		jww.FATAL.Panicf("Failed to delete sent request: %+v", err)
	}
}

// deleteReceiveRequest is a helper function which deletes a Receive request from storage.
func (s *Store) deleteReceiveRequest(r *ReceivedRequest) {
	if err := util.DeleteContact(s.kv, r.partner.ID); err != nil {
		jww.FATAL.Panicf("Failed to delete recieved request "+
			"contact: %+v", err)
	}
}
