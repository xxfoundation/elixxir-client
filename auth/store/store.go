///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
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

	previousNegotiations map[authIdentity]struct{}

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
		previousNegotiations: make(map[authIdentity]struct{}),
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
		previousNegotiations: make(map[authIdentity]struct{}),
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
	if _, ok := s.receivedByID[aid]; ok {
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
	if _, ok := s.sentByID[aih]; ok {
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

// HandleReceivedRequest handles the request singly, only a single operator
// operates on the same request at a time. It will delete the request if no
// error is returned from the handler
func (s *Store) HandleReceivedRequest(partner, myID *id.ID, handler func(*ReceivedRequest) error) error {
	aid := makeAuthIdentity(partner, myID)

	s.mux.RLock()
	rr, ok := s.receivedByID[aid]
	s.mux.RUnlock()

	if !ok {
		return errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	// Take the lock to ensure there is only one operator at a time
	rr.mux.Lock()
	defer rr.mux.Unlock()

	// Check that the request still exists; it could have been deleted while the
	// lock was taken
	s.mux.RLock()
	_, ok = s.receivedByID[aid]
	s.mux.RUnlock()

	if !ok {
		return errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	//run the handler
	handleErr := handler(rr)

	if handleErr != nil {
		return errors.WithMessage(handleErr, "Received error from handler")
	}

	delete(s.receivedByID, aid)
	rr.delete()

	return nil
}

// HandleSentRequest handles the request singly, only a single operator
// operates on the same request at a time. It will delete the request if no
// error is returned from the handler
func (s *Store) HandleSentRequest(partner, myID *id.ID, handler func(request *SentRequest) error) error {
	aid := makeAuthIdentity(partner, myID)

	s.mux.RLock()
	sr, ok := s.sentByID[aid]
	s.mux.RUnlock()

	if !ok {
		return errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	// Take the lock to ensure there is only one operator at a time
	sr.mux.Lock()
	defer sr.mux.Unlock()

	// Check that the request still exists; it could have been deleted while the
	// lock was taken
	s.mux.RLock()
	_, ok = s.sentByID[aid]
	s.mux.RUnlock()

	if !ok {
		return errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	//run the handler
	handleErr := handler(sr)

	if handleErr != nil {
		return errors.WithMessage(handleErr, "Received error from handler")
	}

	delete(s.receivedByID, aid)
	sr.delete()

	return nil
}

// GetReceivedRequest returns the contact representing the partner request
// if it exists. It does not take the lock. It is only meant to return the
// contact to an external API user.
func (s *Store) GetReceivedRequest(partner, myID *id.ID) (contact.Contact, error) {
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
