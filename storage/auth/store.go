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

	mux sync.RWMutex
}

// NewStore creates a new store. All passed in private keys are added as
// sentByFingerprints so they can be used to trigger receivedByID.
func NewStore(kv *versioned.KV, grp *cyclic.Group) error {
	kv = kv.Prefix(storePrefix)
	s := &Store{
		kv:                   kv,
		grp:                  grp,
		receivedByID:         make(map[authIdentity]*ReceivedRequest),
		sentByID:             make(map[authIdentity]*SentRequest),
		previousNegotiations: make(map[id.ID]struct{}),
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
func LoadStore(kv *versioned.KV, defaultID *id.ID, grp *cyclic.Group) (*Store, error) {
	kv = kv.Prefix(storePrefix)

	s := &Store{
		kv:                   kv,
		grp:                  grp,
		receivedByID:         make(map[authIdentity]*ReceivedRequest),
		sentByID:             make(map[authIdentity]*SentRequest),
		previousNegotiations: make(map[id.ID]struct{}),
		defaultID:            defaultID,
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
	fp format.Fingerprint) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	aid := makeAuthIdentity(partner, myID)

	if _, ok := s.sentByID[aid]; ok {
		return errors.Errorf("Cannot make new sentRequest for partner "+
			"%s, one already exists", partner)
	}

	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partner,
		partnerHistoricalPubKey: partnerHistoricalPubKey,
		myPrivKey:               myPrivKey,
		myPubKey:                myPubKey,
		mySidHPubKeyA:           sidHPubA,
		mySidHPrivKeyA:          sidHPrivA,
		fingerprint:             fp,
	}

	if err := sr.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request for partner %s", partner)
	}

	return nil
}

func (s *Store) AddReceived(c contact.Contact, key *sidh.PublicKey) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("AddReceived new contact: %s", c.ID)
	if _, ok := s.receivedByID[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}

	if err := util.StoreContact(s.kv, c); err != nil {
		jww.FATAL.Panicf("Failed to save contact for partner %s", c.ID.String())
	}

	storeKey := util.MakeSIDHPublicKeyKey(c.ID)
	if err := util.StoreSIDHPublicKey(s.kv, key, storeKey); err != nil {
		jww.FATAL.Panicf("Failed to save contact pubKey for partner %s",
			c.ID.String())
	}

	r := &ReceivedRequest{
		rt:               Receive,
		sent:             nil,
		partner:          &c,
		theirSidHPubKeyA: key,
		mux:              sync.Mutex{},
	}

	s.receivedByID[*c.ID] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partner %s", c.ID)
	}

	return nil
}

// GetAllReceived returns all pending received contact receivedByID from storage.
func (s *Store) GetAllReceived() []contact.Contact {
	s.mux.RLock()
	defer s.mux.RUnlock()
	cList := make([]contact.Contact, 0, len(s.receivedByID))
	for key := range s.receivedByID {
		r := s.receivedByID[key]
		if r.rt == Receive {
			cList = append(cList, *r.partner)
		}
	}
	return cList
}

// GetFingerprint can return either a private key or a sentRequest if the
// fingerprint is found. If it returns a sentRequest, then it takes the lock to
// ensure there is only one operator at a time. The user of the API must release
// the lock by calling store.delete() or store.Failed() with the partner ID.
func (s *Store) GetFingerprint(fp format.Fingerprint) (FingerprintType,
	*SentRequest, *cyclic.Int, error) {

	s.mux.RLock()
	r, ok := s.sentByFingerprints[fp]
	s.mux.RUnlock()
	if !ok {
		return 0, nil, nil, errors.Errorf("Fingerprint cannot be found: %v", fp)
	}

	switch r.Type {
	// If it is general, then just return the private key
	case General:
		return General, nil, r.PrivKey, nil

	// If it is specific, then take the request lock and return it
	case Specific:
		r.Request.mux.Lock()
		// Check that the request still exists; it could have been deleted
		// while the lock was taken
		s.mux.RLock()
		_, ok := s.receivedByID[*r.Request.sent.partner]
		s.mux.RUnlock()
		if !ok {
			r.Request.mux.Unlock()
			return 0, nil, nil, errors.Errorf("request associated with "+
				"fingerprint cannot be found: %s", fp)
		}
		// Return the request
		return Specific, r.Request.sent, nil, nil

	default:
		jww.WARN.Printf("Auth request message ignored due to unknown "+
			"fingerprint type %d on lookup; should be impossible", r.Type)
		return 0, nil, nil, errors.New("Unknown fingerprint type")
	}
}

// GetReceivedRequest returns the contact representing the partner request, if
// it exists. If it returns, then it takes the lock to ensure that there is only
// one operator at a time. The user of the API must release the lock by calling
// store.delete() or store.Failed() with the partner ID.
func (s *Store) GetReceivedRequest(partner *id.ID) (contact.Contact, *sidh.PublicKey, error) {
	s.mux.RLock()
	r, ok := s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, nil, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	// Take the lock to ensure there is only one operator at a time
	r.mux.Lock()

	// Check that the request still exists; it could have been deleted while the
	// lock was taken
	s.mux.RLock()
	_, ok = s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok {
		r.mux.Unlock()
		return contact.Contact{}, nil, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.partner, r.theirSidHPubKeyA, nil
}

// GetReceivedRequestData returns the contact representing the partner request
// if it exists. It does not take the lock. It is only meant to return the
// contact to an external API user.
func (s *Store) GetReceivedRequestData(partner *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok || r.partner == nil {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.partner, nil
}

// GetRequest returns request with its type and data. The lock is not taken.
func (s *Store) GetRequest(partner *id.ID) (RequestType, *SentRequest, contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok {
		return 0, nil, contact.Contact{}, errors.New(NoRequest)
	}

	switch r.rt {
	case Sent:
		return Sent, r.sent, contact.Contact{}, nil
	case Receive:
		return Receive, nil, *r.partner, nil
	default:
		return 0, nil, contact.Contact{},
			errors.Errorf("invalid Tag: %d", r.rt)
	}
}

// Done is one of two calls after using a request. This one is to be used when
// the use is unsuccessful. It will allow any thread waiting on access to
// continue using the structure.
// It does not return an error because an error is not handleable.
func (s *Store) Done(partner *id.ID) {
	s.mux.RLock()
	r, ok := s.receivedByID[*partner]
	s.mux.RUnlock()

	if !ok {
		jww.ERROR.Panicf("Request cannot be finished, not "+
			"found: %s", partner)
		return
	}

	r.mux.Unlock()
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
