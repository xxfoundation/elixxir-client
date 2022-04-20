package ud

import (
	"encoding/base64"
	"fmt"
	"gitlab.com/elixxir/crypto/contact"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
	"time"
)

// Happy path.
func TestManager_Search(t *testing.T) {

	m, tnm := newTestManager(t)
	// Generate callback function
	callbackChan := make(chan struct {
		c   []contact.Contact
		err error
	})
	callback := func(c []contact.Contact, err error) {
		callbackChan <- struct {
			c   []contact.Contact
			err error
		}{c: c, err: err}
	}

	// Generate fact list
	var factList fact.FactList
	for i := 0; i < 10; i++ {
		factList = append(factList, fact.Fact{
			Fact: fmt.Sprintf("fact %d", i),
			T:    fact.FactType(rand.Intn(4)),
		})
	}
	factHashes, _ := hashFactList(factList)

	grp := getGroup()

	dhKeyBytes, err := base64.StdEncoding.DecodeString(dhKeyEnc)
	if err != nil {
		panic("Failed to decode dh key")
	}
	dhKey := grp.NewIntFromBytes(dhKeyBytes)

	dhPubKey := dh.GeneratePublicKey(dhKey, grp)

	rng := NewPrng(42)
	privKeyBytes, err := csprng.GenerateInGroup(
		grp.GetP().Bytes(), grp.GetP().ByteLen(), rng)
	if err != nil {
		t.Fatalf("Failed to gen pk bytes: %+v", err)
	}
	privKey := grp.NewIntFromBytes(privKeyBytes)
	dhKeyGenerated := grp.Exp(dhPubKey, privKey, grp.NewInt(1))

	tnm.msg.SetMac(singleUse.MakeMAC(singleUse.NewResponseKey(dhKeyGenerated, 0), tnm.msg.GetContents()))

	var contacts []*Contact
	for _, hash := range factHashes {
		contacts = append(contacts, &Contact{
			UserID:    id.NewIdFromString("user", id.User, t).Marshal(),
			PubKey:    dhPubKey.Bytes(),
			TrigFacts: []*HashFact{hash},
		})
	}

	udContact, err := m.GetContact()
	if err != nil {
		t.Fatalf("Failed to get ud contact: %+v", err)
	}
	prng := NewPrng(42)

	_, _, err = Search(m.network, m.events, prng, m.e2e.GetGroup(),
		udContact, callback, factList, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Search() returned an error: %+v", err)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err != nil {
			t.Fatalf("Callback returned an error: %+v", cb.err)
		}

		c, err := m.GetContact()
		if err != nil {
			t.Errorf("Failed to get UD contact: %+v", err)
		}

		expectedContacts := []contact.Contact{c}
		if !contact.Equal(expectedContacts[0], cb.c[0]) {
			t.Errorf("Failed to get expected Contacts."+
				"\n\texpected: %+v\n\treceived: %+v", expectedContacts, cb.c)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback not called.")
	}
}

// todo; note this was commented out in release
// // Error path: the callback returns an error.
// func TestManager_Search_CallbackError(t *testing.T) {
// 	isReg := uint32(1)
// 	// Set up manager
// 	m := &Manager{
// 		rng:        fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
// 		grp:        cyclic.NewGroup(large.NewInt(107), large.NewInt(2)),
// 		storage:    storage.InitTestingSession(t),
// 		udContact:  contact.Contact{ID: &id.UDB},
// 		net:        newTestNetworkManager(t),
// 		registered: &isReg,
// 	}
//
// 	// Generate callback function
// 	callbackChan := make(chan struct {
// 		c   []contact.Contact
// 		err error
// 	})
// 	callback := func(c []contact.Contact, err error) {
// 		callbackChan <- struct {
// 			c   []contact.Contact
// 			err error
// 		}{c: c, err: err}
// 	}
//
// 	// Generate fact list
// 	factList := fact.FactList{
// 		{Fact: "fact1", T: fact.Username},
// 		{Fact: "fact2", T: fact.Email},
// 		{Fact: "fact3", T: fact.Phone},
// 	}
//
// 	// Trigger lookup response chan
// 	// go func() {
// 	// 	time.Sleep(1 * time.Millisecond)
// 	// 	m.inProgressSearch[0] <- &SearchResponse{
// 	// 		Contacts: nil,
// 	// 		Error:    "Error",
// 	// 	}
// 	// }()
//
// 	// Run the search
// 	err := m.Search(factList, callback, 10*time.Millisecond)
// 	if err != nil {
// 		t.Errorf("Search() returned an error: %+v", err)
// 	}
//
// 	// Verify the callback is called
// 	select {
// 	case cb := <-callbackChan:
// 		if cb.err == nil {
// 			t.Error("Callback did not return an expected error.")
// 		}
//
// 		if cb.c != nil {
// 			t.Errorf("Failed to get expected Contacts."+
// 				"\n\texpected: %v\n\treceived: %v", nil, cb.c)
// 		}
// 	case <-time.After(100 * time.Millisecond):
// 		t.Error("Callback not called.")
// 	}
//
// 	// if _, exists := m.inProgressSearch[m.commID-1]; exists {
// 	// 	t.Error("Failed to delete SearchResponse from inProgressSearch.")
// 	// }
// }
//
// todo; note this was commented out in release
// // Error path: the round event chan times out.
// func TestManager_Search_EventChanTimeout(t *testing.T) {
// 	isReg := uint32(1)
// 	// Set up manager
// 	m := &State{
// 		rng:        fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
// 		grp:        cyclic.NewGroup(large.NewInt(107), large.NewInt(2)),
// 		storage:    storage.InitTestingSession(t),
// 		udContact:  contact.Contact{ID: &id.UDB},
// 		net:        newTestNetworkManager(t),
// 		registered: &isReg,
// 	}
//
// 	// Generate callback function
// 	callbackChan := make(chan struct {
// 		c   []contact.Contact
// 		err error
// 	})
// 	callback := func(c []contact.Contact, err error) {
// 		callbackChan <- struct {
// 			c   []contact.Contact
// 			err error
// 		}{c: c, err: err}
// 	}
//
// 	// Generate fact list
// 	factList := fact.FactList{
// 		{Fact: "fact1", T: fact.Username},
// 		{Fact: "fact2", T: fact.Email},
// 		{Fact: "fact3", T: fact.Phone},
// 	}
//
// 	// Run the search
// 	err := m.Search(factList, callback, 10*time.Millisecond)
// 	if err != nil {
// 		t.Errorf("Search() returned an error: %+v", err)
// 	}
//
// 	// Verify the callback is called
// 	select {
// 	case cb := <-callbackChan:
// 		if cb.err == nil {
// 			t.Error("Callback did not return an expected error.")
// 		}
//
// 		if cb.c != nil {
// 			t.Errorf("Failed to get expected Contacts."+
// 				"\n\texpected: %v\n\treceived: %v", nil, cb.c)
// 		}
// 	case <-time.After(100 * time.Millisecond):
// 		t.Error("Callback not called.")
// 	}
//
// 	// if _, exists := m.inProgressSearch[m.commID-1]; exists {
// 	// 	t.Error("Failed to delete SearchResponse from inProgressSearch.")
// 	// }
// }

// todo: this was not commented out and should be fixed
// Happy path.
//func TestManager_searchResponseHandler(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	callbackChan := make(chan struct {
//		c   []contact.Contact
//		err error
//	})
//	callback := func(c []contact.Contact, err error) {
//		callbackChan <- struct {
//			c   []contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	// Generate fact list
//	var factList fact.FactList
//	for i := 0; i < 10; i++ {
//		factList = append(factList, fact.Fact{
//			Fact: fmt.Sprintf("fact %d", i),
//			T:    fact.FactType(rand.Intn(4)),
//		})
//	}
//	factHashes, factMap := hashFactList(factList)
//
//	var contacts []*Contact
//	var expectedContacts []contact.Contact
//	for i, hash := range factHashes {
//		contacts = append(contacts, &Contact{
//			UserID:    id.NewIdFromString("user", id.User, t).Marshal(),
//			PubKey:    []byte{byte(i + 1)},
//			TrigFacts: []*HashFact{hash},
//		})
//		expectedContacts = append(expectedContacts, contact.Contact{
//			ID:       id.NewIdFromString("user", id.User, t),
//			DhPubKey: m.grp.NewIntFromBytes([]byte{byte(i + 1)}),
//			Facts:    fact.FactList{factMap[string(hash.Hash)]},
//		})
//	}
//
//	// Generate expected Send message
//	payload, err := proto.Marshal(&SearchResponse{Contacts: contacts})
//	if err != nil {
//		t.Fatalf("Failed to marshal LookupSend: %+v", err)
//	}
//
//	m.searchResponseHandler(factMap, callback, payload, nil)
//
//	select {
//	case results := <-callbackChan:
//		if results.err != nil {
//			t.Errorf("Callback returned an error: %+v", results.err)
//		}
//		if !reflect.DeepEqual(expectedContacts, results.c) {
//			t.Errorf("Callback returned incorrect Contacts."+
//				"\nexpected: %+v\nreceived: %+v", expectedContacts, results.c)
//		}
//	case <-time.NewTimer(50 * time.Millisecond).C:
//		t.Error("Callback time out.")
//	}
//}
//
//// Happy path: error is returned on callback when passed into function.
//func TestManager_searchResponseHandler_CallbackError(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	callbackChan := make(chan struct {
//		c   []contact.Contact
//		err error
//	})
//	callback := func(c []contact.Contact, err error) {
//		callbackChan <- struct {
//			c   []contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	testErr := errors.New("search failure")
//
//	m.searchResponseHandler(map[string]fact.Fact{}, callback, []byte{}, testErr)
//
//	select {
//	case results := <-callbackChan:
//		if results.err == nil || !strings.Contains(results.err.Error(), testErr.Error()) {
//			t.Errorf("Callback failed to return error."+
//				"\nexpected: %+v\nreceived: %+v", testErr, results.err)
//		}
//	case <-time.NewTimer(50 * time.Millisecond).C:
//		t.Error("Callback time out.")
//	}
//}
//
//// Error path: SearchResponse message contains an error.
//func TestManager_searchResponseHandler_MessageError(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	callbackChan := make(chan struct {
//		c   []contact.Contact
//		err error
//	})
//	callback := func(c []contact.Contact, err error) {
//		callbackChan <- struct {
//			c   []contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	// Generate expected Send message
//	testErr := "SearchResponse error occurred"
//	payload, err := proto.Marshal(&SearchResponse{Error: testErr})
//	if err != nil {
//		t.Fatalf("Failed to marshal LookupSend: %+v", err)
//	}
//
//	m.searchResponseHandler(map[string]fact.Fact{}, callback, payload, nil)
//
//	select {
//	case results := <-callbackChan:
//		if results.err == nil || !strings.Contains(results.err.Error(), testErr) {
//			t.Errorf("Callback failed to return error."+
//				"\nexpected: %s\nreceived: %+v", testErr, results.err)
//		}
//	case <-time.NewTimer(50 * time.Millisecond).C:
//		t.Error("Callback time out.")
//	}
//}
//
//// Error path: contact is malformed and cannot be parsed.
//func TestManager_searchResponseHandler_ParseContactError(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	callbackChan := make(chan struct {
//		c   []contact.Contact
//		err error
//	})
//	callback := func(c []contact.Contact, err error) {
//		callbackChan <- struct {
//			c   []contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	var contacts []*Contact
//	for i := 0; i < 10; i++ {
//		contacts = append(contacts, &Contact{
//			UserID: []byte{byte(i + 1)},
//		})
//	}
//
//	// Generate expected Send message
//	payload, err := proto.Marshal(&SearchResponse{Contacts: contacts})
//	if err != nil {
//		t.Fatalf("Failed to marshal LookupSend: %+v", err)
//	}
//
//	m.searchResponseHandler(nil, callback, payload, nil)
//
//	select {
//	case results := <-callbackChan:
//		if results.err == nil || !strings.Contains(results.err.Error(), "failed to parse Contact user ID") {
//			t.Errorf("Callback failed to return error: %+v", results.err)
//		}
//	case <-time.NewTimer(50 * time.Millisecond).C:
//		t.Error("Callback time out.")
//	}
//}
//
//// Happy path.
//func Test_hashFactList(t *testing.T) {
//	var factList fact.FactList
//	var expectedHashFacts []*HashFact
//	expectedHashMap := make(map[string]fact.Fact)
//	for i := 0; i < 10; i++ {
//		f := fact.Fact{
//			Fact: fmt.Sprintf("fact %d", i),
//			T:    fact.FactType(rand.Intn(4)),
//		}
//		factList = append(factList, f)
//		expectedHashFacts = append(expectedHashFacts, &HashFact{
//			Hash: factID.Fingerprint(f),
//			Type: int32(f.T),
//		})
//		expectedHashMap[string(factID.Fingerprint(f))] = f
//	}
//
//	hashFacts, hashMap := hashFactList(factList)
//
//	if !reflect.DeepEqual(expectedHashFacts, hashFacts) {
//		t.Errorf("hashFactList() failed to return the expected hash facts."+
//			"\nexpected: %+v\nreceived: %+v", expectedHashFacts, hashFacts)
//	}
//
//	if !reflect.DeepEqual(expectedHashMap, hashMap) {
//		t.Errorf("hashFactList() failed to return the expected hash map."+
//			"\nexpected: %+v\nreceived: %+v", expectedHashMap, hashMap)
//	}
//}
//
//// Happy path.
//func TestManager_parseContacts(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	// Generate fact list
//	var factList fact.FactList
//	for i := 0; i < 10; i++ {
//		factList = append(factList, fact.Fact{
//			Fact: fmt.Sprintf("fact %d", i),
//			T:    fact.FactType(rand.Intn(4)),
//		})
//	}
//	factHashes, factMap := hashFactList(factList)
//
//	var contacts []*Contact
//	var expectedContacts []contact.Contact
//	for i, hash := range factHashes {
//		contacts = append(contacts, &Contact{
//			UserID:    id.NewIdFromString("user", id.User, t).Marshal(),
//			PubKey:    []byte{byte(i + 1)},
//			TrigFacts: []*HashFact{hash},
//		})
//		expectedContacts = append(expectedContacts, contact.Contact{
//			ID:       id.NewIdFromString("user", id.User, t),
//			DhPubKey: m.grp.NewIntFromBytes([]byte{byte(i + 1)}),
//			Facts:    fact.FactList{factMap[string(hash.Hash)]},
//		})
//	}
//
//	testContacts, err := m.parseContacts(contacts, factMap)
//	if err != nil {
//		t.Errorf("parseContacts() returned an error: %+v", err)
//	}
//
//	if !reflect.DeepEqual(expectedContacts, testContacts) {
//		t.Errorf("parseContacts() did not return the expected contacts."+
//			"\nexpected: %+v\nreceived: %+v", expectedContacts, testContacts)
//	}
//}
//
//func TestManager_parseContacts_username(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	// Generate fact list
//	var factList fact.FactList
//	for i := 0; i < 10; i++ {
//		factList = append(factList, fact.Fact{
//			Fact: fmt.Sprintf("fact %d", i),
//			T:    fact.FactType(rand.Intn(4)),
//		})
//	}
//	factHashes, factMap := hashFactList(factList)
//
//	var contacts []*Contact
//	var expectedContacts []contact.Contact
//	for i, hash := range factHashes {
//		contacts = append(contacts, &Contact{
//			UserID:    id.NewIdFromString("user", id.User, t).Marshal(),
//			Username:  "zezima",
//			PubKey:    []byte{byte(i + 1)},
//			TrigFacts: []*HashFact{hash},
//		})
//		expectedContacts = append(expectedContacts, contact.Contact{
//			ID:       id.NewIdFromString("user", id.User, t),
//			DhPubKey: m.grp.NewIntFromBytes([]byte{byte(i + 1)}),
//			Facts:    fact.FactList{{"zezima", fact.Username}, factMap[string(hash.Hash)]},
//		})
//	}
//
//	testContacts, err := m.parseContacts(contacts, factMap)
//	if err != nil {
//		t.Errorf("parseContacts() returned an error: %+v", err)
//	}
//
//	if !reflect.DeepEqual(expectedContacts, testContacts) {
//		t.Errorf("parseContacts() did not return the expected contacts."+
//			"\nexpected: %+v\nreceived: %+v", expectedContacts, testContacts)
//	}
//}
//
//// Error path: provided contact IDs are malformed and cannot be unmarshaled.
//func TestManager_parseContacts_IdUnmarshalError(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//	contacts := []*Contact{{UserID: []byte("invalid ID")}}
//
//	_, err := m.parseContacts(contacts, nil)
//	if err == nil || !strings.Contains(err.Error(), "failed to parse Contact user ID") {
//		t.Errorf("parseContacts() did not return an error when IDs are invalid: %+v", err)
//	}
//}
//
//// mockSingleSearch is used to test the search function, which uses the single-
//// use manager. It adheres to the SingleInterface interface.
//type mockSingleSearch struct {
//}
//
//func (s *mockSingleSearch) TransmitSingleUse(partner contact.Contact, payload []byte,
//	_ string, _ uint8, callback single.ReplyCallback, _ time.Duration) error {
//
//	searchMsg := &SearchSend{}
//	if err := proto.Unmarshal(payload, searchMsg); err != nil {
//		return errors.Errorf("Failed to unmarshal SearchSend: %+v", err)
//	}
//
//	searchResponse := &SearchResponse{
//		Contacts: []*Contact{{
//			UserID: partner.ID.Marshal(),
//			PubKey: partner.DhPubKey.Bytes(),
//		}},
//	}
//	msg, err := proto.Marshal(searchResponse)
//	if err != nil {
//		return errors.Errorf("Failed to marshal SearchResponse: %+v", err)
//	}
//
//	callback(msg, nil)
//	return nil
//}
//
//func (s *mockSingleSearch) StartProcesses() (stoppable.Stoppable, error) {
//	return stoppable.NewSingle(""), nil
//}
