package auth

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const NoRequest = "Request Not Found"

const storePrefix = "requestMap"
const requestMapKey = "map"

const requestMapVersion = 0

type Store struct {
	kv           *versioned.KV
	grp          *cyclic.Group
	requests     map[id.ID]*request
	fingerprints map[format.Fingerprint]fingerprint
	mux          sync.RWMutex
}

// creates a new store. all passed in private keys are added as fingerprints so
// they can be used to trigger requests
func NewStore(kv *versioned.KV, grp *cyclic.Group, privKeys []*cyclic.Int) (*Store, error) {

	kv = kv.Prefix(storePrefix)
	s := &Store{
		kv:           kv,
		grp:          grp,
		requests:     make(map[id.ID]*request),
		fingerprints: make(map[format.Fingerprint]fingerprint),
	}

	for _, key := range privKeys {
		fp := auth.MakeRequestFingerprint(key)
		s.fingerprints[fp] = fingerprint{
			Type:    General,
			PrivKey: key,
			Request: nil,
		}
	}
	return s, s.save()
}

// loads an extant new store. all passed in private keys are added as
// fingerprints so they can be used to trigger requests
func LoadStore(kv *versioned.KV, grp *cyclic.Group, privKeys []*cyclic.Int) (*Store, error) {
	kv = kv.Prefix(storePrefix)
	sentObj, err := kv.Get(requestMapKey)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to Load "+
			"requestMap")
	}

	s := &Store{
		kv:       kv,
		grp:      grp,
		requests: make(map[id.ID]*request),
	}

	for _, key := range privKeys {
		fp := auth.MakeRequestFingerprint(key)
		s.fingerprints[fp] = fingerprint{
			Type:    General,
			PrivKey: key,
			Request: nil,
		}
	}

	var requestList []requestDisk
	if err := json.Unmarshal(sentObj.Data, &requestList); err != nil {
		return nil, errors.WithMessagef(err, "Failed to "+
			"Unmarshal SentRequestMap")
	}

	for _, rDisk := range requestList {

		r := &request{
			rt: RequestType(rDisk.T),
		}

		partner, err := id.Unmarshal(rDisk.ID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load stored id: %+v", err)
		}

		switch r.rt {
		case Sent:
			sr, err := loadSentRequest(kv, partner, grp)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored sentRequest: %+v",
					err)
			}
			r.sent = sr
			s.fingerprints[sr.fingerprint] = fingerprint{
				Type:    Specific,
				PrivKey: nil,
				Request: r,
			}
		case Receive:
			c, err := utility.LoadContact(kv, partner)
			if err != nil {
				jww.FATAL.Panicf("Failed to load stored contact for: %+v",
					err)
			}

			r.receive = &c
		default:
			jww.FATAL.Panicf("Unknown request type: %d", r.rt)
		}
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
		Timestamp: time.Now(),
		Data:      data,
	}

	return s.kv.Set(requestMapKey, &obj)
}

func (s *Store) AddSent(partner *id.ID, myPrivKey, myPubKey *cyclic.Int,
	fp format.Fingerprint) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.requests[*partner]; ok {
		return errors.Errorf("Cannot make new sentRequest for partner "+
			"%s, one already exists", partner)
	}

	sr := &SentRequest{
		kv:          s.kv,
		partner:     partner,
		myPrivKey:   myPrivKey,
		myPubKey:    myPubKey,
		fingerprint: fp,
	}

	if err := sr.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request for partenr %s",
			partner)
	}

	r := &request{
		rt:      Sent,
		sent:    sr,
		receive: nil,
		mux:     sync.Mutex{},
	}

	s.requests[*partner] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding"+
			" partern %s", partner)
	}
	return nil
}

func (s *Store) AddReceived(c contact.Contact) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.requests[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}

	if err := utility.StoreContact(s.kv, c); err != nil {
		jww.FATAL.Panicf("Failed to save contact for partenr %s",
			c.ID)
	}

	r := &request{
		rt:      Receive,
		sent:    nil,
		receive: &c,
		mux:     sync.Mutex{},
	}

	s.requests[*c.ID] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding"+
			" partern %s", c.ID)
	}

	return nil
}

// Can return either a private key or a sentRequest if the fingerprint is found
// if it returns a sentRequest, it takes the lock to ensure there is only one
// operator at a time. The user of the API must release the lock by calling
// store.Delete() or store.Failed() with the partner ID
func (s *Store) GetFingerprint(fp format.Fingerprint) (FingerprintType,
	*SentRequest, *cyclic.Int, error) {

	s.mux.RLock()
	r, ok := s.fingerprints[fp]
	s.mux.RUnlock()
	if !ok {
		return 0, nil, nil,
			errors.Errorf("Fingerprint cannot be found: %s", fp)
	}

	switch r.Type {
	//if its general, just return the private key
	case General:
		return General, nil, r.PrivKey, nil
	//if specific, take the request lock and return it
	case Specific:
		r.Request.mux.Lock()
		//check that the request still exists, it could have been deleted
		//while the lock was taken
		s.mux.RLock()
		_, ok := s.requests[*r.Request.receive.ID]
		s.mux.RUnlock()
		if !ok {
			return 0, nil, nil, errors.Errorf("Fingerprint cannot be "+
				"found: %s", fp)
		}
		//return the request
		return Specific, r.Request.sent, nil, nil
	default:
		return 0, nil, nil, errors.Errorf("Unknown fingerprint type")
	}
}

// returns the contact representing the receive request if it exists.
// if it returns, it takes the lock to ensure there is only one
// operator at a time. The user of the API must release the lock by calling
// store.Delete() or store.Failed() with the partner ID
func (s *Store) GetReceivedRequest(partner *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	//take the lock to ensure there is only one operator at a time
	r.mux.Lock()

	//check that the request still exists, it could have been deleted
	//while the lock was taken
	s.mux.RLock()
	_, ok = s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.receive, nil
}

// returns the contact representing the receive request if it exists.
// Does not take the lock, is only meant to return the contact to an external
// API user
func (s *Store) GetReceivedRequestData(partner *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	r, ok := s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		return contact.Contact{}, errors.Errorf("Received request not "+
			"found: %s", partner)
	}

	return *r.receive, nil
}

// returns request with its type and data. the lock is not taken.
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
			errors.Errorf("invalid Type: %s", r.rt)
	}
}


// One of two calls after using a request. This one to be used when the use
// is unsuccessful. It will allow any thread waiting on access to continue
// using the structure
func (s *Store) Fail(partner *id.ID) error {
	s.mux.RLock()
	r, ok := s.requests[*partner]
	s.mux.RUnlock()

	if !ok {
		return errors.Errorf("Request not found: %s", partner)
	}

	r.mux.Unlock()
	return nil
}

// One of two calls after using a request. This one to be used when the use
// is unsuccessful. It deletes all references to the request associated with the
// passed partner if it exists. It will allow any thread waiting on access to
// continue. They should fail due to the deletion of the structure
func (s *Store) Delete(partner *id.ID) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	r, ok := s.requests[*partner]

	if !ok {
		return errors.Errorf("Request not found: %s", partner)
	}

	switch r.rt {
	case Sent:
		delete(s.fingerprints, r.sent.fingerprint)
		if err := r.sent.delete(); err != nil {
			jww.FATAL.Panicf("Failed to delete sent request: %+v", err)
		}
	case Receive:
		if err := utility.DeleteContact(s.kv, r.receive.ID); err != nil {
			jww.FATAL.Panicf("Failed to delete recieved request "+
				"contact: %+v", err)
		}
	}
	delete(s.requests, *partner)
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to store updated request map after "+
			"deletion: %+v", err)
	}
	r.mux.Unlock()
	return nil
}
