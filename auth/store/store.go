////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"encoding/json"
	"sync"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"

	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
)

const storePrefix = "requestMap"
const requestMapKey = "map"

const requestMapVersion = 0

type Store struct {
	kv                     *versioned.KV
	grp                    *cyclic.Group
	receivedByID           map[id.ID]*ReceivedRequest
	receivedByIDLegacySIDH map[id.ID]*ReceivedRequestLegacySIDH
	sentByID               map[id.ID]*SentRequest

	previousNegotiations map[id.ID]bool

	srh SentRequestHandler

	mux sync.RWMutex
}

// NewOrLoadStore loads an extant new store. All passed in private keys are added as
// sentByFingerprints so they can be used to trigger receivedByID.
// If no store can be found, it creates a new one
func NewOrLoadStore(kv *versioned.KV, grp *cyclic.Group, srh SentRequestHandler) (*Store, error) {
	kv = kv.Prefix(storePrefix)

	s := &Store{
		kv:                     kv,
		grp:                    grp,
		receivedByID:           make(map[id.ID]*ReceivedRequest),
		receivedByIDLegacySIDH: make(map[id.ID]*ReceivedRequestLegacySIDH),
		sentByID:               make(map[id.ID]*SentRequest),
		previousNegotiations:   make(map[id.ID]bool),
		srh:                    srh,
	}

	var requestList []requestDisk

	//load all receivedByID
	sentObj, err := kv.Get(requestMapKey, requestMapVersion)
	if err != nil {
		//no store can be found, lets make a new one
		jww.WARN.Printf("No auth store could be found, making a new one")
		s, err := newStore(kv, grp, srh)
		if err != nil {
			return nil, errors.WithMessagef(err, "Failed to load requestMap")
		}
		return s, nil
	}

	if err := json.Unmarshal(sentObj.Data, &requestList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to "+
			"unmarshal SentRequestMap")
	}

	jww.TRACE.Printf("%d found when loading AuthStore, prefix %s",
		len(requestList), kv.GetPrefix())

	for _, rDisk := range requestList {

		requestType := RequestType(rDisk.T)

		partner, err := id.Unmarshal(rDisk.ID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load stored id: %+v", err)
		}

		switch requestType {
		case Sent:
			sr, err := loadSentRequest(kv, partner, grp)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored sentRequest: %+v", err)
			}

			s.sentByID[*sr.GetPartner()] = sr
			s.srh.Add(sr)
		case Receive:
			rr, err := loadReceivedRequest(kv, partner)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored receivedRequest: %+v", err)
			}

			s.receivedByID[*rr.GetContact().ID] = rr

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
	for _, rr := range s.receivedByID {
		rDisk := requestDisk{
			T:  uint(rr.getType()),
			ID: rr.partner.ID.Marshal(),
		}
		requestIDList = append(requestIDList, rDisk)
	}

	for _, rr := range s.receivedByIDLegacySIDH {
		rDisk := requestDisk{
			T:  uint(rr.getType()),
			ID: rr.partner.ID.Marshal(),
		}
		requestIDList = append(requestIDList, rDisk)
	}

	for _, sr := range s.sentByID {
		rDisk := requestDisk{
			T:  uint(sr.getType()),
			ID: sr.partner.Marshal(),
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

	return s.kv.Set(requestMapKey, &obj)
}

// NewStore creates a new store. All passed in private keys are added as
// sentByFingerprints so they can be used to trigger receivedByID.
func newStore(kv *versioned.KV, grp *cyclic.Group, srh SentRequestHandler) (
	*Store, error) {
	s := &Store{
		kv:                     kv,
		grp:                    grp,
		receivedByID:           make(map[id.ID]*ReceivedRequest),
		receivedByIDLegacySIDH: make(map[id.ID]*ReceivedRequestLegacySIDH),
		sentByID:               make(map[id.ID]*SentRequest),
		previousNegotiations:   make(map[id.ID]bool),
		srh:                    srh,
	}

	err := s.savePreviousNegotiations()
	if err != nil {
		return nil, errors.Errorf(
			"failed to save previousNegotiations partners: %+v", err)
	}

	return s, s.save()
}

func (s *Store) AddSent(partner *id.ID, partnerHistoricalPubKey, myPrivKey,
	myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey,
	sidHPubA *sidh.PublicKey, fp format.Fingerprint,
	reset bool) (*SentRequest, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if !reset {
		if sentRq, ok := s.sentByID[*partner]; ok {
			return sentRq, errors.Errorf("sent request "+
				"already exists for partner %s",
				partner)
		}
		if _, ok := s.receivedByID[*partner]; ok {
			return nil, errors.Errorf("received request "+
				"already exists for partner %s",
				partner)
		}
	}

	sr, err := newSentRequest(s.kv, partner, partnerHistoricalPubKey,
		myPrivKey, myPubKey, sidHPrivA, sidHPubA, fp, reset)

	if err != nil {
		return nil, err
	}

	s.sentByID[*sr.GetPartner()] = sr
	s.srh.Add(sr)
	if err = s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after "+
			"adding partner %s", partner)
	}

	return sr, nil
}

func (s *Store) AddReceived(c contact.Contact, key nike.PublicKey,
	round rounds.Round) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("AddReceived new contact: %s, prefix: %s",
		c.ID, s.kv.GetPrefix())

	if _, ok := s.receivedByID[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}
	if _, ok := s.sentByID[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}
	r := newReceivedRequest(s.kv, c, key, round)

	s.receivedByID[*r.GetContact().ID] = r

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partner %s", c.ID)
	}

	return nil
}

// HandleReceivedRequest handles the request singly, only a single operator
// operates on the same request at a time. It will delete the request if no
// error is returned from the handler
func (s *Store) HandleReceivedRequest(partner *id.ID, handler func(*ReceivedRequest) error) error {

	s.mux.RLock()
	rr, ok := s.receivedByID[*partner]
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
	_, ok = s.receivedByID[*partner]
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

	delete(s.receivedByID, *partner)
	err := s.save()
	rr.delete()

	return err
}

// HandleSentRequest handles the request singly, only a single operator
// operates on the same request at a time. It will delete the request if no
// error is returned from the handler
func (s *Store) HandleSentRequest(partner *id.ID, handler func(request *SentRequest) error) error {

	s.mux.RLock()
	sr, ok := s.sentByID[*partner]
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
	_, ok = s.sentByID[*partner]
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

	delete(s.sentByID, *partner)
	err := s.save()
	sr.delete()

	return err
}

// GetReceivedRequest returns the contact representing the partner request
// if it exists. It does not take the lock. It is only meant to return the
// contact to an external API user.
func (s *Store) GetReceivedRequest(partner *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return r.partner, nil
}

// GetAllReceivedRequests returns a slice of all recieved requests.
func (s *Store) GetAllReceivedRequests() []*ReceivedRequest {

	s.mux.RLock()
	rr := make([]*ReceivedRequest, 0, len(s.receivedByID))

	for _, r := range s.receivedByID {
		rr = append(rr, r)
	}
	s.mux.RUnlock()

	return rr
}
