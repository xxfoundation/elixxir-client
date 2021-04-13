///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
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
		rq, ok := store.fingerprints[auth.MakeRequestFingerprint(pubKeys[i])]
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
	s, kv, privKeys := makeTestStore(t)

	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 id.NewIdFromUInt(rand.Uint64(), id.User, t),
		partnerHistoricalPubKey: s.grp.NewInt(5),
		myPrivKey:               s.grp.NewInt(6),
		myPubKey:                s.grp.NewInt(7),
		fingerprint:             format.Fingerprint{42},
	}

	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() produced an error: %+v", err)
	}

	store, err := LoadStore(kv, s.grp, privKeys)
	if err != nil {
		t.Errorf("LoadStore() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(s, store) {
		t.Errorf("LoadStore() returned incorrect Store."+
			"\n\texpected: %+v\n\treceived: %+v", s, store)
	}
}

// Happy path: tests that the correct SentRequest is added to the map.
func TestStore_AddSent(t *testing.T) {
	s, _, _ := makeTestStore(t)

	partner := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 partner,
		partnerHistoricalPubKey: s.grp.NewInt(5),
		myPrivKey:               s.grp.NewInt(6),
		myPubKey:                s.grp.NewInt(7),
		fingerprint:             format.Fingerprint{42},
	}
	expectedFP := fingerprint{
		Type:    Specific,
		PrivKey: nil,
		Request: &request{Sent, sr, nil, sync.Mutex{}},
	}

	err := s.AddSent(partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.fingerprint)
	if err != nil {
		t.Errorf("AddSent() produced an error: %+v", err)
	}

	if s.requests[*partner] == nil {
		t.Errorf("AddSent() failed to add request to map for partner ID %s.",
			partner)
	} else if !reflect.DeepEqual(sr, s.requests[*partner].sent) {
		t.Errorf("AddSent() failed store the correct SentRequest."+
			"\n\texpected: %+v\n\treceived: %+v",
			sr, s.requests[*partner].sent)
	}

	if _, exists := s.fingerprints[sr.fingerprint]; !exists {
		t.Errorf("AddSent() failed to add fingerprint to map for fingerprint %s.",
			sr.fingerprint)
	} else if !reflect.DeepEqual(expectedFP, s.fingerprints[sr.fingerprint]) {
		t.Errorf("AddSent() failed store the correct fingerprint."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedFP, s.fingerprints[sr.fingerprint])
	}
}

// Error path: request with request already exists in map.
func TestStore_AddSent_PartnerAlreadyExistsError(t *testing.T) {
	s, _, _ := makeTestStore(t)

	partner := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	err := s.AddSent(partner, s.grp.NewInt(5), s.grp.NewInt(6), s.grp.NewInt(7), format.Fingerprint{42})
	if err != nil {
		t.Errorf("AddSent() produced an error: %+v", err)
	}

	err = s.AddSent(partner, s.grp.NewInt(5), s.grp.NewInt(6), s.grp.NewInt(7), format.Fingerprint{42})
	if err == nil {
		t.Errorf("AddSent() did not produce the expected error for a request " +
			"that already exists.")
	}
}

// Happy path.
func TestStore_AddReceived(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}

	err := s.AddReceived(c)
	if err != nil {
		t.Errorf("AddReceived() returned an error: %+v", err)
	}

	if s.requests[*c.ID] == nil {
		t.Errorf("AddReceived() failed to add request to map for partner ID %s.",
			c.ID)
	} else if !reflect.DeepEqual(c, *s.requests[*c.ID].receive) {
		t.Errorf("AddReceived() failed store the correct Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, *s.requests[*c.ID].receive)
	}
}

// Error path: request with request already exists in map.
func TestStore_AddReceived_PartnerAlreadyExistsError(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}

	err := s.AddReceived(c)
	if err != nil {
		t.Errorf("AddReceived() returned an error: %+v", err)
	}

	err = s.AddReceived(c)
	if err == nil {
		t.Errorf("AddReceived() did not produce the expected error for a " +
			"request that already exists.")
	}
}

// Happy path: fingerprints type is General.
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
			"\n\texpected: %s\n\treceived: %s", privKeys[0].Text(10), key.Text(10))
	}
}

// Happy path: fingerprints type is Specific.
func TestStore_GetFingerprint_SpecificFingerprintType(t *testing.T) {
	s, _, _ := makeTestStore(t)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 id.NewIdFromUInt(rand.Uint64(), id.User, t),
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.fingerprint); err != nil {
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
	s.fingerprints[fp] = fingerprint{
		Type:    0,
		PrivKey: s.fingerprints[fp].PrivKey,
		Request: s.fingerprints[fp].Request,
	}
	fpType, request, key, err := s.GetFingerprint(fp)
	if err == nil {
		t.Error("GetFingerprint() did not return an error when the " +
			"FingerprintType is invalid.")
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

// Happy path.
func TestStore_GetReceivedRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	testC, err := s.GetReceivedRequest(c.ID)
	if err != nil {
		t.Errorf("GetReceivedRequest() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(c, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", c, testC)
	}

	// Check if the request's mutex is locked
	if reflect.ValueOf(&s.requests[*c.ID].mux).Elem().FieldByName("state").Int() != 1 {
		t.Errorf("GetReceivedRequest() did not lock mutex.")
	}
}

// Error path: request is deleted between first and second check.
func TestStore_GetReceivedRequest_RequestDeleted(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}

	r := s.requests[*c.ID]
	r.mux.Lock()

	go func() {
		delete(s.requests, *c.ID)
		r.mux.Unlock()
	}()

	testC, err := s.GetReceivedRequest(c.ID)
	if err == nil {
		t.Errorf("GetReceivedRequest() did not return an error when the " +
			"request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, testC)
	}

	// Check if the request's mutex is locked
	if reflect.ValueOf(&r.mux).Elem().FieldByName("state").Int() != 0 {
		t.Errorf("GetReceivedRequest() did not unlock mutex.")
	}
}

// Error path: request does not exist.
func TestStore_GetReceivedRequest_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	testC, err := s.GetReceivedRequest(id.NewIdFromUInt(rand.Uint64(), id.User, t))
	if err == nil {
		t.Errorf("GetReceivedRequest() did not return an error when the " +
			"request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequest() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, testC)
	}
}

// Happy path.
func TestStore_GetReceivedRequestData(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
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

	testC, err := s.GetReceivedRequestData(id.NewIdFromUInt(rand.Uint64(), id.User, t))
	if err == nil {
		t.Errorf("GetReceivedRequestData() did not return an error when the " +
			"request should not exist.")
	}

	if !reflect.DeepEqual(contact.Contact{}, testC) {
		t.Errorf("GetReceivedRequestData() returned incorrect Contact."+
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, testC)
	}
}

// Happy path: request is of type Receive.
func TestStore_GetRequest_ReceiveRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
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
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 id.NewIdFromUInt(rand.Uint64(), id.User, t),
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.fingerprint); err != nil {
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
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, con)
	}
}

// Error path: request type is invalid.
func TestStore_GetRequest_InvalidType(t *testing.T) {
	s, _, _ := makeTestStore(t)
	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	s.requests[*uid] = &request{rt: 42}

	rType, request, con, err := s.GetRequest(uid)
	if err == nil {
		t.Errorf("GetRequest() did not return an error when the request " +
			"type should be invalid.")
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
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, con)
	}
}

// Error path: request does not exist in map.
func TestStore_GetRequest_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)
	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	rType, request, con, err := s.GetRequest(uid)
	if err == nil {
		t.Errorf("GetRequest() did not return an error when the request " +
			"was not in the map.")
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
			"\n\texpected: %+v\n\treceived: %+v", contact.Contact{}, con)
	}
}

// Happy path.
func TestStore_Fail(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("The code did not panic")
		}
	}()

	s.Fail(c.ID)

	// Check if the request's mutex is locked
	if reflect.ValueOf(&s.requests[*c.ID].mux).Elem().FieldByName("state").Int() != 0 {
		t.Errorf("Fail() did not unlock mutex.")
	}
}

// Error path: request does not exist.
func TestStore_Fail_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Fail() did not panic when the request is not in map.")
		}
	}()

	s.Fail(id.NewIdFromUInt(rand.Uint64(), id.User, t))
}

// Happy path: receive request.
func TestStore_Delete_ReceiveRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	c := contact.Contact{ID: id.NewIdFromUInt(rand.Uint64(), id.User, t)}
	if err := s.AddReceived(c); err != nil {
		t.Fatalf("AddReceived() returned an error: %+v", err)
	}
	if _, err := s.GetReceivedRequest(c.ID); err != nil {
		t.Fatalf("GetReceivedRequest() returned an error: %+v", err)
	}

	err := s.Delete(c.ID)
	if err != nil {
		t.Errorf("delete() returned an error: %+v", err)
	}

	if s.requests[*c.ID] != nil {
		t.Errorf("delete() failed to delete request for user %s.", c.ID)
	}
}

// Happy path: sent request.
func TestStore_Delete_SentRequest(t *testing.T) {
	s, _, _ := makeTestStore(t)
	sr := &SentRequest{
		kv:                      s.kv,
		partner:                 id.NewIdFromUInt(rand.Uint64(), id.User, t),
		partnerHistoricalPubKey: s.grp.NewInt(1),
		myPrivKey:               s.grp.NewInt(2),
		myPubKey:                s.grp.NewInt(3),
		fingerprint:             format.Fingerprint{5},
	}
	if err := s.AddSent(sr.partner, sr.partnerHistoricalPubKey, sr.myPrivKey,
		sr.myPubKey, sr.fingerprint); err != nil {
		t.Fatalf("AddSent() returned an error: %+v", err)
	}
	if _, _, _, err := s.GetFingerprint(sr.fingerprint); err != nil {
		t.Fatalf("GetFingerprint() returned an error: %+v", err)
	}

	err := s.Delete(sr.partner)
	if err != nil {
		t.Errorf("delete() returned an error: %+v", err)
	}

	if s.requests[*sr.partner] != nil {
		t.Errorf("delete() failed to delete request for user %s.", sr.partner)
	}

	if _, exists := s.fingerprints[sr.fingerprint]; exists {
		t.Errorf("delete() failed to delete fingerprint for fp %v.", sr.fingerprint)
	}
}

// Error path: request does not exist.
func TestStore_Delete_RequestNotInMap(t *testing.T) {
	s, _, _ := makeTestStore(t)

	err := s.Delete(id.NewIdFromUInt(rand.Uint64(), id.User, t))
	if err == nil {
		t.Errorf("delete() did not return an error when the request was not " +
			"in the map.")
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
