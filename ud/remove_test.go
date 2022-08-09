package ud

//type testRFC struct{}
//
//func (rFC *testRFC) SendRemoveFact(*connect.Host, *pb.FactRemovalRequest) (
//	*messages.Ack, error) {
//	return &messages.Ack{}, nil
//}
//
//func TestRemoveFact(t *testing.T) {
//	storageSess := storage.InitTestingSession(t)
//
//	kv := versioned.NewKV(ekv.Memstore{})
//	udStore, err := store.NewOrLoadStore(kv)
//	if err != nil {
//		t.Fatalf("Failed to initialize store %v", err)
//	}
//
//	// Create our Manager object
//	m := &Manager{
//		services: newTestNetworkManager(t),
//		e2e:      mockE2e{},
//		events:   event.NewEventManager(),
//		user:     storageSess,
//		comms:    mockComms{},
//		store:    udStore,
//		kv:       kv,
//		rng:      fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
//	}
//	f := fact.Fact{
//		Fact: "testing",
//		T:    2,
//	}
//
//	// Set up storage for expected state
//	confirmId := "test"
//	if err = m.store.StoreUnconfirmedFact(confirmId, f); err != nil {
//		t.Fatalf("StoreUnconfirmedFact error: %v", err)
//	}
//
//	if err = m.store.ConfirmFact(confirmId); err != nil {
//		t.Fatalf("ConfirmFact error: %v", err)
//	}
//
//	tRFC := testRFC{}
//
//	err = m.removeFact(f, &tRFC)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
//
//func (rFC *testRFC) SendRemoveUser(*connect.Host, *pb.FactRemovalRequest) (
//	*messages.Ack, error) {
//	return &messages.Ack{}, nil
//}
//
//func TestRemoveUser(t *testing.T) {
//
//	storageSess := storage.InitTestingSession(t)
//
//	kv := versioned.NewKV(ekv.Memstore{})
//	udStore, err := store.NewOrLoadStore(kv)
//	if err != nil {
//		t.Fatalf("Failed to initialize store %v", err)
//	}
//
//	mockId := id.NewIdFromBytes([]byte("test"), t)
//
//	// Create our Manager object
//	m := &Manager{
//		services: newTestNetworkManager(t),
//		e2e:      mockE2e{},
//		events:   event.NewEventManager(),
//		user:     storageSess,
//		comms:    mockComms{},
//		store:    udStore,
//		kv:       kv,
//		rng:      fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
//	}
//
//	f := fact.Fact{
//		Fact: "testing",
//		T:    2,
//	}
//
//	tRFC := testRFC{}
//
//	udHost, err := m.getHost()
//	if err != nil {
//		t.Fatalf("getHost error: %v", err)
//	}
//
//	err = m.permanentDeleteAccount(f, mockId, &tRFC, udHost)
//	if err != nil {
//		t.Fatal(err)
//	}
//}
