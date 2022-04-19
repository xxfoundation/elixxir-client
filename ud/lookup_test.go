package ud

//// Happy path.
//func TestManager_Lookup(t *testing.T) {
//	storageSess := storage.InitTestingSession(t)
//
//	kv := versioned.NewKV(ekv.Memstore{})
//	udStore, err := store.NewOrLoadStore(kv)
//	if err != nil {
//		t.Fatalf("Failed to initialize store %v", err)
//	}
//
//	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
//
//	// Create our Manager object
//	m := Manager{
//		network: newTestNetworkManager(t),
//		e2e:     mockE2e{},
//		events:  event.NewEventManager(),
//		user:    storageSess,
//		comms:   &mockComms{},
//		store:   udStore,
//		kv:      kv,
//		rng:     fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
//	}
//
//	// Generate callback function
//	callbackChan := make(chan struct {
//		c   contact.Contact
//		err error
//	})
//	callback := func(c contact.Contact, err error) {
//		callbackChan <- struct {
//			c   contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//	uid := id.NewIdFromUInt(0x500000000000000, id.User, t)
//
//	udContact, err := m.GetContact()
//	if err != nil {
//		t.Fatalf("Failed to get contact: %v", err)
//	}
//
//	r := m.e2e.GetGroup().NewInt(1)
//	m.e2e.GetGroup().Random(r)
//	s := ""
//	jsonable, err := r.MarshalJSON()
//	if err != nil {
//		t.Fatalf("failed to marshal json: %v", err)
//	}
//	for _, b := range jsonable {
//		s += strconv.Itoa(int(b)) + ", "
//	}
//
//	t.Logf("%v", r.Bytes())
//	t.Logf("%s", s)
//
//	// Run the lookup
//	_, _, err = Lookup(m.network, m.rng, grp, udContact, callback, uid, 10*time.Millisecond)
//	if err != nil {
//		t.Errorf("Lookup() returned an error: %+v", err)
//	}
//
//	// Verify the callback is called
//	select {
//	case cb := <-callbackChan:
//		if cb.err != nil {
//			t.Errorf("Callback returned an error: %+v", cb.err)
//		}
//
//		expectedContact := contact.Contact{
//			ID:       uid,
//			DhPubKey: grp.NewIntFromBytes([]byte{5}),
//		}
//		if !reflect.DeepEqual(expectedContact, cb.c) {
//			t.Errorf("Failed to get expected Contact."+
//				"\n\texpected: %v\n\treceived: %v", expectedContact, cb.c)
//		}
//	case <-time.After(100 * time.Millisecond):
//		t.Error("Callback not called.")
//	}
//}

//// Happy path.
//func TestManager_lookupResponseProcess(t *testing.T) {
//	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(2))
//
//	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)
//	callbackChan := make(chan struct {
//		c   contact.Contact
//		err error
//	})
//	callback := func(c contact.Contact, err error) {
//		callbackChan <- struct {
//			c   contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//	pubKey := []byte{5}
//	expectedContact := contact.Contact{
//		ID:       uid,
//		DhPubKey: grp.NewIntFromBytes(pubKey),
//	}
//
//	// Generate expected Send message
//	payload, err := proto.Marshal(&LookupResponse{PubKey: pubKey})
//	if err != nil {
//		t.Fatalf("Failed to marshal LookupSend: %+v", err)
//	}
//
//	(uid, callback, payload, nil)
//
//	select {
//	case results := <-callbackChan:
//		if results.err != nil {
//			t.Errorf("Callback returned an error: %+v", results.err)
//		}
//		if !reflect.DeepEqual(expectedContact, results.c) {
//			t.Errorf("Callback returned unexpected Contact."+
//				"\nexpected: %+v\nreceived: %+v", expectedContact, results.c)
//		}
//	case <-time.NewTimer(50 * time.Millisecond).C:
//		t.Error("Callback time out.")
//	}
//}

//// Happy path: error is returned on callback when passed into function.
//func TestManager_lookupResponseProcess_CallbackError(t *testing.T) {
//	m := &Manager{e2e: mockE2e{}}
//
//	callbackChan := make(chan struct {
//		c   contact.Contact
//		err error
//	})
//	callback := func(c contact.Contact, err error) {
//		callbackChan <- struct {
//			c   contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	testErr := errors.New("lookup failure")
//
//	m.lookupResponseProcess(nil, callback, []byte{}, testErr)
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

//// Error path: LookupResponse message contains an error.
//func TestManager_lookupResponseProcess_MessageError(t *testing.T) {
//	m := &Manager{grp: cyclic.NewGroup(large.NewInt(107), large.NewInt(2))}
//
//	uid := id.NewIdFromUInt(rand.Uint64(), id.User, t)
//	callbackChan := make(chan struct {
//		c   contact.Contact
//		err error
//	})
//	callback := func(c contact.Contact, err error) {
//		callbackChan <- struct {
//			c   contact.Contact
//			err error
//		}{c: c, err: err}
//	}
//
//	// Generate expected Send message
//	testErr := "LookupResponse error occurred"
//	payload, err := proto.Marshal(&LookupResponse{Error: testErr})
//	if err != nil {
//		t.Fatalf("Failed to marshal LookupSend: %+v", err)
//	}
//
//	m.lookupResponseProcess(uid, callback, payload, nil)
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

// mockSingleLookup is used to test the lookup function, which uses the single-
// use manager. It adheres to the SingleInterface interface.
type mockSingleLookup struct {
}

//func (s *mockSingleLookup) TransmitSingleUse(_ contact.Contact, payload []byte,
//	_ string, _ uint8, callback single.ReplyCallback, _ time.Duration) error {
//
//	lookupMsg := &LookupSend{}
//	if err := proto.Unmarshal(payload, lookupMsg); err != nil {
//		return errors.Errorf("Failed to unmarshal LookupSend: %+v", err)
//	}
//
//	lr := &LookupResponse{PubKey: lookupMsg.UserID[:1]}
//	msg, err := proto.Marshal(lr)
//	if err != nil {
//		return errors.Errorf("Failed to marshal LookupResponse: %+v", err)
//	}
//
//	callback(msg, nil)
//	return nil
//}
//
//func (s *mockSingleLookup) StartProcesses() (stoppable.Stoppable, error) {
//	return stoppable.NewSingle(""), nil
//}
