package ud

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Happy path.
func TestManager_Search(t *testing.T) {
	// Set up mock UD values
	grp := getGroup()
	prng := NewPrng(42)
	privKeyBytes, err := csprng.GenerateInGroup(
		grp.GetP().Bytes(), grp.GetP().ByteLen(), prng)
	if err != nil {
		t.Fatalf("Failed to generate a mock private key: %v", err)
	}
	udMockPrivKey := grp.NewIntFromBytes(privKeyBytes)

	// Set up mock manager
	m, tnm := newTestManager(t)
	// Generate callback function
	callbackChan := make(mockChannel)
	callback := func(c []contact.Contact, err error) {
		callbackChan <- mockResponse{
			c:   c,
			err: err}
	}

	// Generate fact list
	var factList fact.FactList
	udbId, err := id.Unmarshal(tnm.instance.GetFullNdf().Get().UDB.ID)
	if err != nil {
		t.Fatalf("Failed to unmarshal ID in mock ndf: %v", err)
	}
	factList = append(factList, fact.Fact{
		Fact: udbId.String(),
		T:    fact.Username,
	})

	grp = getGroup()

	var contacts []*Contact
	udContact, err := m.GetContact()
	if err != nil {
		t.Fatalf("Failed to get ud contact: %v", err)
	}
	contacts = append(contacts, &Contact{
		UserID: udContact.ID.Bytes(),
		PubKey: udContact.DhPubKey.Bytes(),
	})

	// Generate a mock UD service to respond to the search request

	receiver := newMockReceiver(callbackChan, contacts, t)

	mockListener := single.Listen(SearchTag, udbId, udMockPrivKey,
		tnm, grp, receiver)
	defer mockListener.Stop()

	timeout := 100 * time.Millisecond
	p := single.RequestParams{
		Timeout:             timeout,
		MaxResponseMessages: 1,
		CmixParams:          cmix.GetDefaultCMIXParams(),
	}

	_, _, err = Search(m.user,
		udContact, callback, factList, p)
	if err != nil {
		t.Fatalf("Search() returned an error: %+v", err)
	}

	// Verify the callback is called
	select {
	case cb := <-callbackChan:
		if cb.err != nil {
			t.Fatalf("Callback returned an error: %+v", cb.err)
		}

		expectedContacts := []contact.Contact{udContact}
		if !contact.Equal(expectedContacts[0], cb.c[0]) {
			t.Errorf("Failed to get expected Contacts."+
				"\n\texpected: %+v\n\treceived: %+v", expectedContacts, cb.c)
		}
	case <-time.After(timeout):
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
