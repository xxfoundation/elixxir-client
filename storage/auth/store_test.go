///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"bytes"
	"github.com/cloudflare/circl/dh/sidh"
	sidhinterface "gitlab.com/elixxir/client/interfaces/sidh"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
)

// Happy path.
func TestNewStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	privKeys := make([]*cyclic.Int, 10)
	pubKeys := make([]*cyclic.Int, 10)
	for i := range privKeys {
		privKeys[i] = grp.NewInt(rand.Int63n(172))
		pubKeys[i] = grp.ExpG(privKeys[i], grp.NewInt(1))
	}

	store, err := NewStore(kv, grp, privKeys)
	if err != nil {
		t.Errorf("NewStore() returned an error: %+v", err)
	}

	for i, key := range privKeys {
		rq, ok := store.sentByFingerprints[auth.MakeRequestFingerprint(pubKeys[i])]
		if !ok {
			t.Errorf("Key not found in map (%d): %s", i, pubKeys[i].Text(16))
		} else if rq.PrivKey.Cmp(key) != 0 {
			t.Errorf("Key found in map (%d) does not match private: "+
				"%s vs %s", i, key.Text(10), rq.PrivKey.Text(10))
		}
	}
}

// Happy path.
func TestLoadStore(t *testing.T) {
	rng := csprng.NewSystemRNG()

	// Create a random storage object + keys
	s, kv, privKeys := makeTestStore(t)

	// Generate random contact information and add it to the store
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	// Create a sent request object and add it to the store
	privSidh, pubSidh := genSidhAKeys(rng)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 id.NewIdFromUInt(rand.Uint64(), id.User, t),
		partnerHistoricalPubKey: s.grp.NewInt(5),
		myPrivKey:               s.grp.NewInt(6),
		myPubKey:                s.grp.NewInt(7),
		mySidHPrivKeyA:          privSidh,
		mySidHPubKeyA:           pubSidh,
		fingerprint:             format.Fingerprint{42},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.mySidHPrivKeyA, sr.mySidHPubKeyA,
		sr.fingerprint); err != nil {
		t.Fatalf("AddSent() produced an error: %+v", err)
	}

	s.AddIfNew(
		sr.partner, auth.CreateNegotiationFingerprint(privKeys[0], sidhPubKey))

	// Attempt to load the store
	store, err := LoadStore(kv, s.grp, privKeys)
	if err != nil {
		t.Errorf("LoadStore() returned an error: %+v", err)
	}

	// Verify what was loaded equals what was put in.
	// if !reflect.DeepEqual(s, store) {
	// 	t.Errorf("LoadStore() returned incorrect Store."+
	// 		"\n\texpected: %+v\n\treceived: %+v", s, store)
	// }

	// The above no longer works, so specifically check for the
	// sent request and contact object that
	// was added.
	testC, testPubKeyA, err := store.GetReceivedRequest(c.ID)
	if err != nil {
		t.Errorf("GetReceivedRequest() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(c, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, testC)
	}

	keyBytes := make([]byte, sidhinterface.PubKeyByteSize)
	sidhPubKey.Export(keyBytes)
	expKeyBytes := make([]byte, sidhinterface.PubKeyByteSize)
	testPubKeyA.Export(expKeyBytes)
	if !reflect.DeepEqual(keyBytes, expKeyBytes) {
		t.Errorf("GetReceivedRequest did not send proper sidh bytes")
	}

	partner := sr.partner
	if s.receivedByID[*partner] == nil {
		t.Errorf("AddSent() failed to add request to map for "+
			"partner ID %s.", partner)
	} else if !reflect.DeepEqual(sr, s.receivedByID[*partner].sent) {
		t.Errorf("AddSent() failed store the correct SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v",
			sr, s.receivedByID[*partner].sent)
	}
	expectedFP := fingerprint{
		Type:    Specific,
		PrivKey: nil,
		Request: &ReceivedRequest{Sent, sr, nil, nil, sync.Mutex{}},
	}
	if _, exists := s.sentByFingerprints[sr.fingerprint]; !exists {
		t.Errorf("AddSent() failed to add fingerprint to map for "+
			"fingerprint %s.", sr.fingerprint)
	} else if !reflect.DeepEqual(expectedFP,
		s.sentByFingerprints[sr.fingerprint]) {
		t.Errorf("AddSent() failed store the correct fingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedFP, s.sentByFingerprints[sr.fingerprint])
	}
}

// Happy path: tests that the correct SentRequest is added to the map.
func TestStore_AddSent(t *testing.T) {
	rng := csprng.NewSystemRNG()
	s, _, _ := makeTestStore(t)

	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)

	partner := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partner,
		partnerHistoricalPubKey: s.grp.NewInt(5),
		myPrivKey:               s.grp.NewInt(6),
		myPubKey:                s.grp.NewInt(7),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{42},
	}
	// Note: nil keys are nil because they are not used when
	// "Sent" sent request object is set.
	// FIXME: We're overloading the same data type with multiple
	// meaning and this is a difficult pattern to debug/implement correctly.
	// Instead, consider separate data structures for different state and
	// crossreferencing and storing separate or "typing" that object when
	// serialized into the same collection.
	expectedFP := fingerprint{
		Type:    Specific,
		PrivKey: nil,
		Request: &ReceivedRequest{Sent, sr, nil, nil, sync.Mutex{}},
	}

	err := s.AddSent(partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.mySidHPrivKeyA, sr.mySidHPubKeyA,
		sr.fingerprint)
	if err != nil {
		t.Errorf("AddSent() produced an error: %+v", err)
	}

	if s.receivedByID[*partner] == nil {
		t.Errorf("AddSent() failed to add request to map for "+
			"partner ID %s.", partner)
	} else if !reflect.DeepEqual(sr, s.receivedByID[*partner].sent) {
		t.Errorf("AddSent() failed store the correct SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v",
			sr, s.receivedByID[*partner].sent)
	}

	if _, exists := s.sentByFingerprints[sr.fingerprint]; !exists {
		t.Errorf("AddSent() failed to add fingerprint to map for "+
			"fingerprint %s.", sr.fingerprint)
	} else if !reflect.DeepEqual(expectedFP,
		s.sentByFingerprints[sr.fingerprint]) {
		t.Errorf("AddSent() failed store the correct fingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedFP, s.sentByFingerprints[sr.fingerprint])
	}
}

// Error path: request with request already exists in map.
func TestStore_AddSent_PartnerAlreadyExistsError(t *testing.T) {
	s, _, _ := makeTestStore(t)

	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)

	partner := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	err := s.AddSent(partner, s.grp.NewInt(5), s.grp.NewInt(6),
		s.grp.NewInt(7), sidhPrivKey, sidhPubKey,
		format.Fingerprint{42})
	if err != nil {
		t.Errorf("AddSent() produced an error: %+v", err)
	}

	err = s.AddSent(partner, s.grp.NewInt(5), s.grp.NewInt(6),
		s.grp.NewInt(7), sidhPrivKey, sidhPubKey,
		format.Fingerprint{42})
	if err == nil {
		t.Errorf("AddSent() did not produce the expected error for " +
			"a request that already exists.")
	}
}

// Happy path.
func TestStore_AddReceived(t *testing.T) {
	s, _, _ := makeTestStore(t)

	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)

	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}

	err := s.AddReceived(c, sidhPubKey)
	if err != nil {
		t.Errorf("AddReceived() returned an error: %+v", err)
	}

	if s.receivedByID[*c.ID] == nil {
		t.Errorf("AddReceived() failed to add request to map for "+
			"partner ID %s.", c.ID)
	} else if !reflect.DeepEqual(c, *s.receivedByID[*c.ID].partner) {
		t.Errorf("AddReceived() failed store the correct Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c,
			*s.receivedByID[*c.ID].partner)
	}
}

// Error path: request with request already exists in map.
func TestStore_AddReceived_PartnerAlreadyExistsError(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}

	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)

	err := s.AddReceived(c, sidhPubKey)
	if err != nil {
		t.Errorf("AddReceived() returned an error: %+v", err)
	}

	err = s.AddReceived(c, sidhPubKey)
	if err == nil {
		t.Errorf("AddReceived() did not produce the expected error " +
			"for a request that already exists.")
	}
}

// Happy path: sentByFingerprints type is General.
func TestStore_GetFingerprint_GeneralFingerprintType(t *testing.T) {
	s, _, privKeys := makeTestStore(t)

	pubkey := s.grp.ExpG(privKeys[0], s.grp.NewInt(1))
	fp := auth.MakeRequestFingerprint(pubkey)
	fpType, request, key, err := s.GetFingerprint(fp)
	if err != nil {
		t.Errorf("GetFingerprint() returned an error: %+v", err)
	}
	if fpType != General {
		t.Errorf("GetFingerprint() returned incorrect FingerprintType."+
			"\n\texpected: %d\n\treceived: %d", General, fpType)
	}
	if request != nil {
		t.Errorf("GetFingerprint() returned incorrect request."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}

	if key.Cmp(privKeys[0]) == -2 {
		t.Errorf("GetFingerprint() returned incorrect key."+
			"\n\texpected: %s\n\treceived: %s",
			privKeys[0].Text(10), key.Text(10))
	}
}

// Happy path: sentByFingerprints type is Specific.
func TestStore_GetFingerprint_SpecificFingerprintType(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)

	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
		sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA, sr.mySidHPubKeyA,
		sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}

	fpType, request, key, err := s.GetFingerprint(sr.fingerprint)
	if err != nil {
		t.Errorf("GetFingerprint() returned an error: %+v", err)
	}
	if fpType != Specific {
		t.Errorf("GetFingerprint() returned incorrect FingerprintType."+
			"\n\texpected: %d\n\treceived: %d", Specific, fpType)
	}
	if request == nil {
		t.Errorf("GetFingerprint() returned incorrect request."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if key != nil {
		t.Errorf("GetFingerprint() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %s", nil, key.Text(10))
	}
}

// Error path: fingerprint does not exist.
func TestStore_GetFingerprint_FingerprintError(t *testing.T) {
	s, _, _ := makeTestStore(t)

	fpType, request, key, err := s.GetFingerprint(format.Fingerprint{32})
	if err == nil {
		t.Error("GetFingerprint() did not return an error when the " +
			"fingerprint should not be found.")
	}
	if fpType != 0 {
		t.Errorf("GetFingerprint() returned incorrect FingerprintType."+
			"\n\texpected: %d\n\treceived: %d", 0, fpType)
	}
	if request != nil {
		t.Errorf("GetFingerprint() returned incorrect request."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if key != nil {
		t.Errorf("GetFingerprint() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v", nil, key)
	}
}

// Error path: fingerprint has an invalid type.
func TestStore_GetFingerprint_InvalidFingerprintType(t *testing.T) {
	s, _, privKeys := makeTestStore(t)

	fp := auth.MakeRequestFingerprint(privKeys[0])
	s.sentByFingerprints[fp] = fingerprint{
		Type:    0,
		PrivKey: s.sentByFingerprints[fp].PrivKey,
		Request: s.sentByFingerprints[fp].Request,
	}
	fpType, request, key, err := s.GetFingerprint(fp)
	if err == nil {
		t.Error("GetFingerprint() did not return an error when the " +
			"FingerprintType is invalid.")
	}
	if fpType != 0 {
		t.Errorf("GetFingerprint() returned incorrect "+
			"FingerprintType.\n\texpected: %d\n\treceived: %d",
			0, fpType)
	}
	if request != nil {
		t.Errorf("GetFingerprint() returned incorrect request."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if key != nil {
		t.Errorf("GetFingerprint() returned incorrect key."+
			"\n\texpected: %v\n\treceived: %v", nil, key)
	}
}

// Happy path.
func TestStore_GetReceivedRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)

	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	testC, testPubKeyA, err := s.GetReceivedRequest(c.ID)
	if err != nil {
		t.Errorf("GetReceivedRequest() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(c, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, testC)
	}

	// Check if the request's mutex is locked
	if reflect.ValueOf(&s.receivedByID[*c.ID].mux).Elem().FieldByName(
		"state").Int() != 1 {
		t.Errorf("GetReceivedRequest() did not lock mutex.")
	}

	keyBytes := make([]byte, sidhinterface.PubKeyByteSize)
	sidhPubKey.Export(keyBytes)
	expKeyBytes := make([]byte, sidhinterface.PubKeyByteSize)
	testPubKeyA.Export(expKeyBytes)
	if !reflect.DeepEqual(keyBytes, expKeyBytes) {
		t.Errorf("GetReceivedRequest did not send proper sidh bytes")
	}
}

// Error path: request is deleted between first and second check.
func TestStore_GetReceivedRequest_RequestDeleted(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	r := s.receivedByID[*c.ID]
	r.mux.Lock()

	go func() {
		delete(s.receivedByID, *c.ID)
		r.mux.Unlock()
	}()

	testC, _, err := s.GetReceivedRequest(c.ID)
	if err == nil {
		t.Errorf("GetReceivedRequest() did not return an error " +
			"when the request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			testC)
	}

	// Check if the request's mutex is locked
	if reflect.ValueOf(&r.mux).Elem().FieldByName("state").Int() != 0 {
		t.Errorf("GetReceivedRequest() did not unlock mutex.")
	}
}

// Error path: request does not exist.
func TestStore_GetReceivedRequest_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	testC, testPubKeyA, err := s.GetReceivedRequest(
		id.NewIdFromUInt(rand.Uint64(),
			id.User, t))
	if err == nil {
		t.Errorf("GetReceivedRequest() did not return an error " +
			"when the request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			testC)
	}

	if testPubKeyA != nil {
		t.Errorf("Expected empty sidh public key!")
	}
}

// Happy path.
func TestStore_GetReceivedRequestData(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	testC, err := s.GetReceivedRequestData(c.ID)
	if err != nil {
		t.Errorf("GetReceivedRequestData() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(c, testC) {
		t.Errorf("GetReceivedRequestData() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, testC)
	}
}

// Error path: request does not exist.
func TestStore_GetReceivedRequestData_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	testC, err := s.GetReceivedRequestData(id.NewIdFromUInt(
		rand.Uint64(),
		id.User, t))
	if err == nil {
		t.Errorf("GetReceivedRequestData() did not return an error " +
			"when the request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequestData() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			testC)
	}
}

// Happy path: request is of type Receive.
func TestStore_GetRequest_ReceiveRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	rType, request, con, err := s.GetRequest(c.ID)
	if err != nil {
		t.Errorf("GetRequest() returned an error: %+v", err)
	}
	if rType != Receive {
		t.Errorf("GetRequest() returned incorrect RequestType."+
			"\n\texpected: %d\n\treceived: %d", Receive, rType)
	}
	if request != nil {
		t.Errorf("GetRequest() returned incorrect SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if !reflect.DeepEqual(c, con) {
		t.Errorf("GetRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, con)
	}
}

// Happy path: request is of type Sent.
func TestStore_GetRequest_SentRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)

	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.mySidHPrivKeyA, sr.mySidHPubKeyA,
		sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}

	rType, request, con, err := s.GetRequest(sr.partner)
	if err != nil {
		t.Errorf("GetRequest() returned an error: %+v", err)
	}
	if rType != Sent {
		t.Errorf("GetRequest() returned incorrect RequestType."+
			"\n\texpected: %d\n\treceived: %d", Sent, rType)
	}
	if !reflect.DeepEqual(sr, request) {
		t.Errorf("GetRequest() returned incorrect SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v", sr, request)
	}
	if !reflect.DeepEqual(contact.Contact{}, con) {
		t.Errorf("GetRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			con)
	}
}

// Error path: request type is invalid.
func TestStore_GetRequest_InvalidType(t *testing.T) {
	s, _, _ := makeTestStore(t)
	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	s.receivedByID[*uid] = &ReceivedRequest{rt: 42}

	rType, request, con, err := s.GetRequest(uid)
	if err == nil {
		t.Errorf("GetRequest() did not return an error " +
			"when the request type should be invalid.")
	}
	if rType != 0 {
		t.Errorf("GetRequest() returned incorrect RequestType."+
			"\n\texpected: %d\n\treceived: %d", 0, rType)
	}
	if request != nil {
		t.Errorf("GetRequest() returned incorrect SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if !reflect.DeepEqual(contact.Contact{}, con) {
		t.Errorf("GetRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			con)
	}
}

// Error path: request does not exist in map.
func TestStore_GetRequest_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)
	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	rType, request, con, err := s.GetRequest(uid)
	if err == nil {
		t.Errorf("GetRequest() did not return an error " +
			"when the request was not in the map.")
	}
	if rType != 0 {
		t.Errorf("GetRequest() returned incorrect RequestType."+
			"\n\texpected: %d\n\treceived: %d", 0, rType)
	}
	if request != nil {
		t.Errorf("GetRequest() returned incorrect SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v", nil, request)
	}
	if !reflect.DeepEqual(contact.Contact{}, con) {
		t.Errorf("GetRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{},
			con)
	}
}

// Happy path.
func TestStore_Fail(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did not panic")
		}
	}()

	s.Done(c.ID)

	// Check if the request's mutex is locked
	if reflect.ValueOf(&s.receivedByID[*c.ID].mux).Elem().FieldByName(
		"state").Int() != 0 {
		t.Errorf("Done() did not unlock mutex.")
	}
}

// Error path: request does not exist.
func TestStore_Fail_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Done() did not panic when the " +
				"request is not in map.")
		}
	}()

	s.Done(id.NewIdFromUInt(rand.Uint64(), id.User, t))
}

// Happy path: partner request.
func TestStore_Delete_ReceiveRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	err := s.Delete(c.ID)
	if err != nil {
		t.Errorf("delete() returned an error: %+v", err)
	}

	if s.receivedByID[*c.ID] != nil {
		t.Errorf("delete() failed to delete request for user %s.", c.ID)
	}
}

// Happy path: sent request.
func TestStore_Delete_SentRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
		sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
		sr.mySidHPubKeyA, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}
	if _, _, _, err := s.GetFingerprint(sr.fingerprint); err != nil {
		t.Fatalf("GetFingerprint() returned an error: %+v", err)
	}

	err := s.Delete(sr.partner)
	if err != nil {
		t.Errorf("delete() returned an error: %+v", err)
	}

	if s.receivedByID[*sr.partner] != nil {
		t.Errorf("delete() failed to delete request for user %s.",
			sr.partner)
	}

	if _, exists := s.sentByFingerprints[sr.fingerprint]; exists {
		t.Errorf("delete() failed to delete fingerprint for fp %v.",
			sr.fingerprint)
	}
}

// Error path: request does not exist.
func TestStore_Delete_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	err := s.Delete(id.NewIdFromUInt(rand.Uint64(), id.User, t))
	if err == nil {
		t.Errorf("delete() did not return an error when the request " +
			"was not in the map.")
	}
}

// Unit test of Store.GetAllReceived.
func TestStore_GetAllReceived(t *testing.T) {
	s, _, _ := makeTestStore(t)
	numReceived := 10

	expectContactList := make([]contact.Contact, 0, numReceived)
	// Add multiple received contact receivedByID
	for i := 0; i < numReceived; i++ {
		c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
		rng := csprng.NewSystemRNG()
		_, sidhPubKey := genSidhAKeys(rng)

		if err := s.AddReceived(c, sidhPubKey); err != nil {
			t.Fatalf("AddReceived() returned an error: %+v", err)
		}

		expectContactList = append(expectContactList, c)
	}

	// Check that GetAllReceived returns all contacts
	receivedContactList := s.GetAllReceived()
	if len(receivedContactList) != numReceived {
		t.Errorf("GetAllReceived did not return expected amount of contacts."+
			"\nExpected: %d"+
			"\nReceived: %d", numReceived, len(receivedContactList))
	}

	// Sort expected and received lists so that they are in the same order
	// since extraction from a map does not maintain order
	sort.Slice(expectContactList, func(i, j int) bool {
		return bytes.Compare(expectContactList[i].ID.Bytes(), expectContactList[j].ID.Bytes()) == -1
	})
	sort.Slice(receivedContactList, func(i, j int) bool {
		return bytes.Compare(receivedContactList[i].ID.Bytes(), receivedContactList[j].ID.Bytes()) == -1
	})

	// Check validity of contacts
	if !reflect.DeepEqual(expectContactList, receivedContactList) {
		t.Errorf("GetAllReceived did not return expected contact list."+
			"\nExpected: %+v"+
			"\nReceived: %+v", expectContactList, receivedContactList)
	}

}

// Tests that Store.GetAllReceived returns an empty list when there are no
// received receivedByID.
func TestStore_GetAllReceived_EmptyList(t *testing.T) {
	s, _, _ := makeTestStore(t)

	// Check that GetAllReceived returns all contacts
	receivedContactList := s.GetAllReceived()
	if len(receivedContactList) != 0 {
		t.Errorf("GetAllReceived did not return expected amount of contacts."+
			"\nExpected: %d"+
			"\nReceived: %d", 0, len(receivedContactList))
	}

	// Add Sent and Receive receivedByID
	for i := 0; i < 10; i++ {
		partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
		rng := csprng.NewSystemRNG()
		sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
		sr := &SentRequest{
			kv:                      s.kv,
			partner:                 partnerID,
			partnerHistoricalPubKey: s.grp.NewInt(1),
			myPrivKey:               s.grp.NewInt(2),
			myPubKey:                s.grp.NewInt(3),
			mySidHPrivKeyA:          sidhPrivKey,
			mySidHPubKeyA:           sidhPubKey,
			fingerprint:             format.Fingerprint{5},
		}
		if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
			sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
			sr.mySidHPubKeyA, sr.fingerprint); err != nil {
			t.Fatalf("AddSent() returned an error: %+v", err)
		}
	}

	// Check that GetAllReceived returns all contacts
	receivedContactList = s.GetAllReceived()
	if len(receivedContactList) != 0 {
		t.Errorf("GetAllReceived did not return expected amount of contacts. "+
			"It may be pulling from Sent Requests."+
			"\nExpected: %d"+
			"\nReceived: %d", 0, len(receivedContactList))
	}

}

// Tests that Store.GetAllReceived returns only Sent receivedByID when there
// are both Sent and Receive receivedByID in Store.
func TestStore_GetAllReceived_MixSentReceived(t *testing.T) {
	s, _, _ := makeTestStore(t)
	numReceived := 10

	// Add multiple received contact receivedByID
	for i := 0; i < numReceived; i++ {
		// Add received request
		c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
		rng := csprng.NewSystemRNG()
		_, sidhPubKey := genSidhAKeys(rng)

		if err := s.AddReceived(c, sidhPubKey); err != nil {
			t.Fatalf("AddReceived() returned an error: %+v", err)
		}

		// Add sent request
		partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
		sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
		sr := &SentRequest{
			kv:                      s.kv,
			partner:                 partnerID,
			partnerHistoricalPubKey: s.grp.NewInt(1),
			myPrivKey:               s.grp.NewInt(2),
			myPubKey:                s.grp.NewInt(3),
			mySidHPrivKeyA:          sidhPrivKey,
			mySidHPubKeyA:           sidhPubKey,
			fingerprint:             format.Fingerprint{5},
		}
		if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
			sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
			sr.mySidHPubKeyA, sr.fingerprint); err != nil {
			t.Fatalf("AddSent() returned an error: %+v", err)
		}
	}

	// Check that GetAllReceived returns all contacts
	receivedContactList := s.GetAllReceived()
	if len(receivedContactList) != numReceived {
		t.Errorf("GetAllReceived did not return expected amount of contacts. "+
			"It may be pulling from Sent Requests."+
			"\nExpected: %d"+
			"\nReceived: %d", numReceived, len(receivedContactList))
	}

}

// Error case: Call DeleteRequest on a request that does
// not exist.
func TestStore_DeleteRequest_NonexistantRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	err := s.DeleteRequest(c.ID)
	if err != nil {
		t.Errorf("DeleteRequest should return an error " +
			"when trying to delete a partner request")
	}

}

// Unit test.
func TestStore_DeleteReceiveRequests(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	err := s.DeleteReceiveRequests()
	if err != nil {
		t.Fatalf("DeleteReceiveRequests returned an error: %+v", err)
	}

	if s.receivedByID[*c.ID] != nil {
		t.Errorf("delete() failed to delete request for user %s.", c.ID)
	}
}

// Unit test.
func TestStore_DeleteSentRequests(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
		sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
		sr.mySidHPubKeyA, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}

	err := s.DeleteSentRequests()
	if err != nil {
		t.Fatalf("DeleteSentRequests returned an error: %+v", err)
	}

	if s.receivedByID[*sr.partner] != nil {
		t.Errorf("delete() failed to delete request for user %s.",
			sr.partner)
	}

	if _, exists := s.sentByFingerprints[sr.fingerprint]; exists {
		t.Errorf("delete() failed to delete fingerprint for fp %v.",
			sr.fingerprint)
	}
}

// Tests that DeleteSentRequests does not affect partner receivedByID in map
func TestStore_DeleteSentRequests_ReceiveInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	rng := csprng.NewSystemRNG()
	_, sidhPubKey := genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	err := s.DeleteSentRequests()
	if err != nil {
		t.Fatalf("DeleteSentRequests returned an error: %+v", err)
	}

	if s.receivedByID[*c.ID] == nil {
		t.Fatalf("DeleteSentRequests removes partner receivedByID!")
	}

}

// Tests that DeleteReceiveRequests does not affect sent receivedByID in map
func TestStore_DeleteReceiveRequests_SentInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
		sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
		sr.mySidHPubKeyA, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}

	err := s.DeleteReceiveRequests()
	if err != nil {
		t.Fatalf("DeleteSentRequests returned an error: %+v", err)
	}

	if s.receivedByID[*partnerID] == nil {
		t.Fatalf("DeleteReceiveRequests removes sent receivedByID!")
	}

}

// Unit test.
func TestStore_DeleteAllRequests(t *testing.T) {
	s, _, _ := makeTestStore(t)
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	rng := csprng.NewSystemRNG()
	sidhPrivKey, sidhPubKey := genSidhAKeys(rng)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partnerID,
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		mySidHPrivKeyA:          sidhPrivKey,
		mySidHPubKeyA:           sidhPubKey,
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey,
		sr.myPrivKey, sr.myPubKey, sr.mySidHPrivKeyA,
		sr.mySidHPubKeyA, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}

	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	_, sidhPubKey = genSidhAKeys(rng)
	if err := s.AddReceived(c, sidhPubKey); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	err := s.DeleteAllRequests()
	if err != nil {
		t.Fatalf("DeleteAllRequests returned an error: %+v", err)
	}

	if s.receivedByID[*sr.partner] != nil {
		t.Errorf("delete() failed to delete request for user %s.",
			sr.partner)
	}

	if _, exists := s.sentByFingerprints[sr.fingerprint]; exists {
		t.Errorf("delete() failed to delete fingerprint for fp %v.",
			sr.fingerprint)
	}

	if s.receivedByID[*c.ID] != nil {
		t.Errorf("delete() failed to delete request for user %s.", c.ID)
	}

}

func makeTestStore(t *testing.T) (*Store, *versioned.KV, []*cyclic.Int) {
	kv := versioned.NewKV(make(ekv.Memstore))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(0))
	privKeys := make([]*cyclic.Int, 10)
	for i := range privKeys {
		privKeys[i] = grp.NewInt(rand.Int63n(170) + 1)
	}

	store, err := NewStore(kv, grp, privKeys)
	if err != nil {
		t.Fatalf("Failed to create new Store: %+v", err)
	}

	return store, kv, privKeys
}

func genSidhAKeys(rng io.Reader) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKeyA := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	sidHPubKeyA := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)

	if err := sidHPrivKeyA.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKeyA.GeneratePublicKey(sidHPubKeyA)

	return sidHPrivKeyA, sidHPubKeyA
}

func genSidhBKeys(rng io.Reader) (*sidh.PrivateKey, *sidh.PublicKey) {
	sidHPrivKeyB := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	sidHPubKeyB := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)

	if err := sidHPrivKeyB.Generate(rng); err != nil {
		panic("failure to generate SidH A private key")
	}
	sidHPrivKeyB.GeneratePublicKey(sidHPubKeyB)

	return sidHPrivKeyB, sidHPubKeyB
}
