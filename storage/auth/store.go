///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"encoding/json"
	"sync"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const NoRequest = "Request Not Found"

const storePrefix = "requestMap"
const requestMapKey = "map"

const requestMapVersion = 0

type Store struct {
	kv                   *versioned.KV
	grp                  *cyclic.Group
	requests             map[id.ID]*request
	fingerprints         map[format.Fingerprint]fingerprint
	previousNegotiations map[id.ID]struct{}
	mux                  sync.RWMutex
}

// NewStore creates a new store. All passed in private keys are added as
// fingerprints so they can be used to trigger requests.
func NewStore(kv *versioned.KV, grp *cyclic.Group, privKeys []*cyclic.Int) (*Store, error) {
	kv = kv.Prefix(storePrefix)
	s := &Store{
		kv:                   kv,
		grp:                  grp,
		requests:             make(map[id.ID]*request),
		fingerprints:         make(map[format.Fingerprint]fingerprint),
		previousNegotiations: make(map[id.ID]struct{}),
	}

	for _, key := range privKeys {
		pubkey := grp.ExpG(key, grp.NewInt(1))
		fp := auth.MakeRequestFingerprint(pubkey)
		s.fingerprints[fp] = fingerprint{
			Type:    General,
			PrivKey: key,
			Request: nil,
		}
	}

	err := s.savePreviousNegotiations()
	if err != nil {
		return nil, errors.Errorf(
			"failed to load previousNegotiations partners: %+v", err)
	}

	return s, s.save()
}

// LoadStore loads an extant new store. All passed in private keys are added as
// fingerprints so they can be used to trigger requests.
func LoadStore(kv *versioned.KV, grp *cyclic.Group, privKeys []*cyclic.Int) (*Store, error) {
	kv = kv.Prefix(storePrefix)
	sentObj, err := kv.Get(requestMapKey, requestMapVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load requestMap")
	}

	s := &Store{
		kv:                   kv,
		grp:                  grp,
		requests:             make(map[id.ID]*request),
		fingerprints:         make(map[format.Fingerprint]fingerprint),
		previousNegotiations: make(map[id.ID]struct{}),
	}

	for _, key := range privKeys {
		pubkey := grp.ExpG(key, grp.NewInt(1))
		fp := auth.MakeRequestFingerprint(pubkey)
		s.fingerprints[fp] = fingerprint{
			Type:    General,
			PrivKey: key,
			Request: nil,
		}
	}

	var requestList []requestDisk
	if err := json.Unmarshal(sentObj.Data, &requestList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to "+
			"unmarshal SentRequestMap")
	}
	jww.TRACE.Printf("%d found when loading AuthStore", len(requestList))
	for _, rDisk := range requestList {
		r := &request{
			rt: RequestType(rDisk.T),
		}

		var rid *id.ID

		partner, err := id.Unmarshal(rDisk.ID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load stored id: %+v", err)
		}

		switch r.rt {
		case Sent:
			sr, err := loadSentRequest(kv, partner, grp)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored sentRequest: %+v", err)
			}
			r.sent = sr
			s.fingerprints[sr.fingerprint] = fingerprint{
				Type:    Specific,
				PrivKey: nil,
				Request: r,
			}
			rid = sr.partner
			r.sent = sr

		case Receive:
			c, err := util.LoadContact(kv, partner)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored contact for: %+v", err)
			}

			key, err := util.LoadSIDHPublicKey(kv,
				util.MakeSIDHPublicKeyKey(c.ID))
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored contact for: %+v", err)
			}

			rid = c.ID
			r.receive = &c
			r.theirSidHPubKeyA = key

		default:
			jww.FATAL.Panicf("Unknown request type: %d", r.rt)
		}

		// store in the request map
		s.requests[*rid] = r
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
	requestIDList := make([]requestDisk, len(s.requests))
	index := 0
	for pid, r := range s.requests {
		rDisk := requestDisk{
			T:  uint(r.rt),
			ID: pid.Marshal(),
		}
		requestIDList[index] = rDisk
		index++
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

func (s *Store) AddSent(partner *id.ID, partnerHistoricalPubKey, myPrivKey,
	myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey, sidHPubA *sidh.PublicKey,
	fp format.Fingerprint) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.requests[*partner]; ok {
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

	r := &request{
		rt:      Sent,
		sent:    sr,
		receive: nil,
		mux:     sync.Mutex{},
	}

	s.requests[*partner] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partern %s", partner)
	}

	jww.INFO.Printf("AddSent PUBKEY FINGERPRINT: %v", sr.fingerprint)
	jww.INFO.Printf("AddSent PUBKEY: %v", sr.myPubKey.Bytes())
	jww.INFO.Printf("AddSent Partner: %s", partner)

	s.fingerprints[sr.fingerprint] = fingerprint{
		Type:    Specific,
		PrivKey: nil,
		Request: r,
	}

	return nil
}

func (s *Store) AddReceived(c contact.Contact, key *sidh.PublicKey) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("AddReceived new contact: %s", c.ID)
	if _, ok := s.requests[*c.ID]; ok {
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

	r := &request{
		rt:               Receive,
		sent:             nil,
		receive:          &c,
		theirSidHPubKeyA: key,
		mux:              sync.Mutex{},
	}

	s.requests[*c.ID] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partner %s", c.ID)
	}

	return nil
}

// GetAllReceived returns all pending received contact requests from storage.
func (s *Store) GetAllReceived() []contact.Contact {
	s.mux.RLock()
	defer s.mux.RUnlock()
	cList := make([]contact.Contact, 0, len(s.requests))
	for key := range s.requests {
		r := s.requests[key]
		if r.rt == Receive {
			cList = append(cList, *r.receive)
		}
	}
	return cList
}

// GetAllReceived returns all pending received contact requests from storage.
func (s *Store) GetAllSentIDs() []*id.ID {
	s.mux.RLock()
	defer s.mux.RUnlock()
	cList := make([]*id.ID, 0, len(s.requests))
	for key := range s.requests {
		r := s.requests[key]
		if r.rt == Sent {
			cList = append(cList, r.sent.partner)
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
	r, ok := s.fingerprints[fp]
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
		_, ok := s.requests[*r.Request.sent.partner]
		s.mux.RUnlock()
		if !ok {
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

// GetReceivedRequest returns the contact representing the receive request, if
// it exists. If it returns, then it takes the lock to ensure that there is only
// one operator at a time. The user of the API must release the lock by calling
// store.delete() or store.Failed() with the partner ID.
func (s *Store) GetReceivedRequest(partner *id.ID) (contact.Contact, *sidh.PublicKey, error) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
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
	_, ok = s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		r.mux.Unlock()
		return contact.Contact{}, nil, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.receive, r.theirSidHPubKeyA, nil
}

// GetReceivedRequestData returns the contact representing the receive request
// if it exists. It does not take the lock. It is only meant to return the
// contact to an external API user.
func (s *Store) GetReceivedRequestData(partner *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
	s.mux.RUnlock()

	if !ok || r.receive == nil {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.receive, nil
}

// GetRequest returns request with its type and data. The lock is not taken.
func (s *Store) GetRequest(partner *id.ID) (RequestType, *SentRequest, contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		return 0, nil, contact.Contact{}, errors.New(NoRequest)
	}

	switch r.rt {
	case Sent:
		return Sent, r.sent, contact.Contact{}, nil
	case Receive:
		return Receive, nil, *r.receive, nil
	default:
		return 0, nil, contact.Contact{},
			errors.Errorf("invalid Type: %d", r.rt)
	}
}

// Done is one of two calls after using a request. This one is to be used when
// the use is unsuccessful. It will allow any thread waiting on access to
// continue using the structure.
// It does not return an error because an error is not handleable.
func (s *Store) Done(partner *id.ID) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
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
	r, ok := s.requests[*partner]

	if !ok {
		return errors.Errorf("Request not found: %s", partner)
	}

	switch r.rt {
	case Sent:
		s.deleteSentRequest(r)
	case Receive:
		s.deleteReceiveRequest(r)
	}

	delete(s.requests, *partner)
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

	for partnerId, req := range s.requests {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.requests, partnerId)
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.requests, partnerId)
		}

	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all requests: %+v", err)
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

	req, ok := s.requests[*partnerId]
	if !ok {
		return errors.Errorf("Request for %s does not exist", partnerId)
	}

	switch req.rt {
	case Sent:
		s.deleteSentRequest(req)
	case Receive:
		s.deleteReceiveRequest(req)
	}

	delete(s.requests, *partnerId)

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting receive request for partner %s: %+v", partnerId, err)
	}

	return nil
}

// DeleteSentRequests deletes all Sent requests from Store.
func (s *Store) DeleteSentRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.requests {
		switch req.rt {
		case Sent:
			s.deleteSentRequest(req)
			delete(s.requests, partnerId)
		case Receive:
			continue
		}
	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all sent requests: %+v", err)
	}

	return nil
}

// DeleteReceiveRequests deletes all Receive requests from Store.
func (s *Store) DeleteReceiveRequests() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for partnerId, req := range s.requests {
		switch req.rt {
		case Sent:
			continue
		case Receive:
			s.deleteReceiveRequest(req)
			delete(s.requests, partnerId)
		}
	}

	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deleting all receive requests: %+v", err)
	}

	return nil
}

// deleteSentRequest is a helper function which deletes a Sent request from storage.
func (s *Store) deleteSentRequest(r *request) {
	delete(s.fingerprints, r.sent.fingerprint)
	if err := r.sent.delete(); err != nil {
		jww.FATAL.Panicf("Failed to delete sent request: %+v", err)
	}
}

// deleteReceiveRequest is a helper function which deletes a Receive request from storage.
func (s *Store) deleteReceiveRequest(r *request) {
	if err := util.DeleteContact(s.kv, r.receive.ID); err != nil {
		jww.FATAL.Panicf("Failed to delete recieved request "+
			"contact: %+v", err)
	}
}
