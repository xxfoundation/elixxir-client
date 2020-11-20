package ud

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Happy path.
func TestManager_Search(t *testing.T) {
	// Set up manager
	m := &Manager{
		rng:              fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
		grp:              cyclic.NewGroup(large.NewInt(107), large.NewInt(2)),
		storage:          storage.InitTestingSession(t),
		udID:             &id.UDB,
		inProgressSearch: map[uint64]chan *SearchResponse{},
		net:              newTestNetworkManager(t),
	}

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
	factList := fact.FactList{
		{Fact: "fact1", T: fact.Username},
		{Fact: "fact2", T: fact.Email},
		{Fact: "fact3", T: fact.Phone},
	}

	// Trigger lookup response chan
	responseContacts := []*Contact{
		{
			UserID: id.NewIdFromUInt(5, id.User, t).Bytes(),
			PubKey: []byte{42},
			TrigFacts: []*HashFact{
				{Hash: factID.Fingerprint(factList[0]), Type: int32(factList[0].T)},
				{Hash: factID.Fingerprint(factList[1]), Type: int32(factList[1].T)},
				{Hash: factID.Fingerprint(factList[2]), Type: int32(factList[2].T)},
			},
		},
	}
	go func() {
		time.Sleep(1 * time.Millisecond)
		m.inProgressSearch[0] <- &SearchResponse{
			Contacts: responseContacts,
			Error:    "",
		}
	}()

	// Run the search
	err := m.Search(factList, callback, 20*time.Millisecond)
	if err != nil {
		t.Errorf("Search() returned an error: %+v", err)
	}

	// Generate expected Send message
	factHashes, factMap := hashFactList(factList)
	payload, err := proto.Marshal(&SearchSend{
		Fact:   factHashes,
		CommID: m.commID - 1,
	})
	if err != nil {
		t.Fatalf("Failed to marshal SearchSend: %+v", err)
	}
	expectedMsg := message.Send{
		Recipient:   m.udID,
		Payload:     payload,
		MessageType: message.UdSearch,
	}

	// Verify the message is correct
	if !reflect.DeepEqual(expectedMsg, m.net.(*testNetworkManager).msg) {
		t.Errorf("Failed to send correct message."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedMsg, m.net.(*testNetworkManager).msg)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err != nil {
			t.Errorf("Callback returned an error: %+v", cb.err)
		}

		expectedContacts, err := m.parseContacts(responseContacts, factMap)
		if err != nil {
			t.Fatalf("parseResponseContacts() returned an error: %+v", err)
		}
		if !reflect.DeepEqual(expectedContacts, cb.c) {
			t.Errorf("Failed to get expected Contacts."+
				"\n\texpected: %v\n\treceived: %v", expectedContacts, cb.c)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback not called.")
	}

	if _, exists := m.inProgressSearch[m.commID-1]; exists {
		t.Error("Failed to delete SearchResponse from inProgressSearch.")
	}
}

// Error path: the callback returns an error.
func TestManager_Search_CallbackError(t *testing.T) {
	// Set up manager
	m := &Manager{
		rng:              fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
		grp:              cyclic.NewGroup(large.NewInt(107), large.NewInt(2)),
		storage:          storage.InitTestingSession(t),
		udID:             &id.UDB,
		inProgressSearch: map[uint64]chan *SearchResponse{},
		net:              newTestNetworkManager(t),
	}

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
	factList := fact.FactList{
		{Fact: "fact1", T: fact.Username},
		{Fact: "fact2", T: fact.Email},
		{Fact: "fact3", T: fact.Phone},
	}

	// Trigger lookup response chan
	go func() {
		time.Sleep(1 * time.Millisecond)
		m.inProgressSearch[0] <- &SearchResponse{
			Contacts: nil,
			Error:    "Error",
		}
	}()

	// Run the search
	err := m.Search(factList, callback, 10*time.Millisecond)
	if err != nil {
		t.Errorf("Search() returned an error: %+v", err)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err == nil {
			t.Error("Callback did not return an expected error.")
		}

		if cb.c != nil {
			t.Errorf("Failed to get expected Contacts."+
				"\n\texpected: %v\n\treceived: %v", nil, cb.c)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback not called.")
	}

	if _, exists := m.inProgressSearch[m.commID-1]; exists {
		t.Error("Failed to delete SearchResponse from inProgressSearch.")
	}
}

// Error path: the round event chan times out.
func TestManager_Search_EventChanTimeout(t *testing.T) {
	// Set up manager
	m := &Manager{
		rng:              fastRNG.NewStreamGenerator(12, 3, csprng.NewSystemRNG),
		grp:              cyclic.NewGroup(large.NewInt(107), large.NewInt(2)),
		storage:          storage.InitTestingSession(t),
		udID:             &id.UDB,
		inProgressSearch: map[uint64]chan *SearchResponse{},
		net:              newTestNetworkManager(t),
	}

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
	factList := fact.FactList{
		{Fact: "fact1", T: fact.Username},
		{Fact: "fact2", T: fact.Email},
		{Fact: "fact3", T: fact.Phone},
	}

	// Run the search
	err := m.Search(factList, callback, 10*time.Millisecond)
	if err != nil {
		t.Errorf("Search() returned an error: %+v", err)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err == nil {
			t.Error("Callback did not return an expected error.")
		}

		if cb.c != nil {
			t.Errorf("Failed to get expected Contacts."+
				"\n\texpected: %v\n\treceived: %v", nil, cb.c)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Callback not called.")
	}

	if _, exists := m.inProgressSearch[m.commID-1]; exists {
		t.Error("Failed to delete SearchResponse from inProgressSearch.")
	}
}
