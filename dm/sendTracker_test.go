package dm

// type mockClient struct{}

// func (mc *mockClient) GetMaxMessageLength() int {
// 	return 2048
// }
// func (mc *mockClient) SendWithAssembler(*id.ID, cmix.MessageAssembler,
// 	cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
// 	return rounds.Round{}, ephemeral.Id{}, nil
// }
// func (mc *mockClient) IsHealthy() bool {
// 	return true
// }
// func (mc *mockClient) AddIdentity(*id.ID, time.Time, bool, message.Processor)                       {}
// func (mc *mockClient) AddIdentityWithHistory(*id.ID, time.Time, time.Time, bool, message.Processor) {}
// func (mc *mockClient) AddService(*id.ID, message.Service, message.Processor)                        {}
// func (mc *mockClient) DeleteClientService(*id.ID)                                                   {}
// func (mc *mockClient) RemoveIdentity(*id.ID)                                                        {}
// func (mc *mockClient) GetRoundResults(time.Duration, cmix.RoundEventCallback, ...id.Round)          {}
// func (mc *mockClient) AddHealthCallback(func(bool)) uint64                                          { return 0 }
// func (mc *mockClient) RemoveHealthCallback(uint64)                                                  {}

// // Test MessageReceive basic logic.
// func TestSendTracker_MessageReceive(t *testing.T) {
// 	kv := versioned.NewKV(ekv.MakeMemstore())
// 	uuidNum := uint64(0)
// 	rid := id.Round(2)

// 	r := rounds.Round{
// 		ID:         rid,
// 		Timestamps: make(map[states.Round]time.Time),
// 	}
// 	r.Timestamps[states.QUEUED] = time.Now()
// 	trigger := func(chID *id.ID, umi *userMessageInternal, ts time.Time,
// 		receptionID receptionID.EphemeralIdentity, round rounds.Round,
// 		status SentStatus) (uint64, error) {
// 		oldUUID := uuidNum
// 		uuidNum++
// 		return oldUUID, nil
// 	}

// 	updateStatus := func(uuid uint64, messageID cryptoMessage.ID,
// 		timestamp time.Time, round rounds.Round, status SentStatus) {
// 	}

// 	cid := id.NewIdFromString("channel", id.User, t)

// 	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

// 	st := loadSendTracker(&mockClient{}, kv, trigger, nil, updateStatus, crng)

// 	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
// 		[]byte("hello"))
// 	process := st.MessageReceive(mid, r)
// 	if process {
// 		t.Fatalf("Did not receive expected result from MessageReceive")
// 	}

// 	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
// 		userMessage: &UserMessage{},
// 		channelMessage: &ChannelMessage{
// 			Lease:       netTime.Now().UnixNano(),
// 			RoundID:     uint64(rid),
// 			PayloadType: 0,
// 			Payload:     []byte("hello"),
// 		}})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	err = st.send(uuid, mid, rounds.Round{
// 		ID:    rid,
// 		State: 1,
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}
// 	process = st.MessageReceive(mid, r)
// 	if !process {
// 		t.Fatalf("Did not receive expected result from MessageReceive")
// 	}

// 	cid2 := id.NewIdFromString("channel two", id.User, t)
// 	uuid2, err := st.denotePendingSend(cid2, &userMessageInternal{
// 		userMessage: &UserMessage{},
// 		channelMessage: &ChannelMessage{
// 			Lease:       netTime.Now().UnixNano(),
// 			RoundID:     uint64(rid),
// 			PayloadType: 0,
// 			Payload:     []byte("hello again"),
// 		}})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	err = st.send(uuid2, mid, rounds.Round{
// 		ID:    rid,
// 		State: 1,
// 	})
// 	process = st.MessageReceive(mid, r)
// 	if !process {
// 		t.Fatalf("Did not receive expected result from MessageReceive")
// 	}
// }

// // Test failedSend function, confirming that data is stored appropriately and
// // callbacks are called.
// func TestSendTracker_failedSend(t *testing.T) {
// 	triggerCh := make(chan SentStatus)

// 	kv := versioned.NewKV(ekv.MakeMemstore())

// 	adminTrigger := func(chID *id.ID, cm *ChannelMessage, ts time.Time,
// 		messageID cryptoMessage.ID, receptionID receptionID.EphemeralIdentity,
// 		round rounds.Round, status SentStatus) (uint64, error) {
// 		return 0, nil
// 	}

// 	updateStatus := func(uuid uint64, messageID cryptoMessage.ID,
// 		timestamp time.Time, round rounds.Round, status SentStatus) {
// 		triggerCh <- status
// 	}

// 	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

// 	st := loadSendTracker(&mockClient{}, kv, nil, adminTrigger, updateStatus, crng)

// 	cid := id.NewIdFromString("channel", id.User, t)
// 	rid := id.Round(2)
// 	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
// 		[]byte("hello"))
// 	uuid, err := st.denotePendingAdminSend(cid, &ChannelMessage{
// 		Lease:       0,
// 		RoundID:     uint64(rid),
// 		PayloadType: 0,
// 		Payload:     []byte("hello"),
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	err = st.failedSend(uuid)
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	timeout := time.NewTicker(time.Second * 5)
// 	select {
// 	case s := <-triggerCh:
// 		if s != Failed {
// 			t.Fatalf("Did not receive failed from failed message")
// 		}
// 	case <-timeout.C:
// 		t.Fatal("Timed out waiting for trigger chan")
// 	}

// 	trackedRound, ok := st.byRound[rid]
// 	if ok {
// 		t.Fatal("Should not have found a tracked round")
// 	}
// 	if len(trackedRound.List) != 0 {
// 		t.Fatal("Did not find expected number of trackedRounds")
// 	}

// 	_, ok = st.byMessageID[mid]
// 	if ok {
// 		t.Error("Should not have found tracked message")
// 	}

// 	_, ok = st.unsent[uuid]
// 	if ok {
// 		t.Fatal("Should not have found an unsent")
// 	}
// }

// // Test send tracker send function, confirming that data is stored appropriately
// // // and callbacks are called
// func TestSendTracker_send(t *testing.T) {
// 	triggerCh := make(chan bool)

// 	kv := versioned.NewKV(ekv.MakeMemstore())
// 	trigger := func(chID *id.ID, umi *userMessageInternal, ts time.Time,
// 		receptionID receptionID.EphemeralIdentity, round rounds.Round,
// 		status SentStatus) (uint64, error) {
// 		return 0, nil
// 	}

// 	updateStatus := func(uuid uint64, messageID cryptoMessage.ID,
// 		timestamp time.Time, round rounds.Round, status SentStatus) {
// 		triggerCh <- true
// 	}

// 	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

// 	st := loadSendTracker(&mockClient{}, kv, trigger, nil, updateStatus, crng)

// 	cid := id.NewIdFromString("channel", id.User, t)
// 	rid := id.Round(2)
// 	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
// 		[]byte("hello"))
// 	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
// 		userMessage: &UserMessage{},
// 		channelMessage: &ChannelMessage{
// 			Lease:       0,
// 			RoundID:     uint64(rid),
// 			PayloadType: 0,
// 			Payload:     []byte("hello"),
// 		},
// 		messageID: mid,
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	err = st.send(uuid, mid, rounds.Round{
// 		ID:    rid,
// 		State: 2,
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	timeout := time.NewTicker(time.Second * 5)
// 	select {
// 	case <-triggerCh:
// 	case <-timeout.C:
// 		t.Fatal("Timed out waiting for trigger chan")
// 	}

// 	trackedRound, ok := st.byRound[rid]
// 	if !ok {
// 		t.Fatal("Should have found a tracked round")
// 	}
// 	if len(trackedRound.List) != 1 {
// 		t.Fatal("Did not find expected number of trackedRounds")
// 	}
// 	if trackedRound.List[0].MsgID != mid {
// 		t.Fatalf("Did not find expected message ID in trackedRounds")
// 	}

// 	trackedMsg, ok := st.byMessageID[mid]
// 	if !ok {
// 		t.Error("Should have found tracked message")
// 	}
// 	if trackedMsg.MsgID != mid {
// 		t.Fatalf("Did not find expected message ID in byMessageID")
// 	}
// }

// // Test loading stored byRound map from storage.
// func TestSendTracker_load_store(t *testing.T) {
// 	kv := versioned.NewKV(ekv.MakeMemstore())

// 	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

// 	st := loadSendTracker(&mockClient{}, kv, nil, nil, nil, crng)
// 	cid := id.NewIdFromString("channel", id.User, t)
// 	rid := id.Round(2)
// 	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
// 		[]byte("hello"))
// 	st.byRound[rid] = trackedList{
// 		List:           []*tracked{{MsgID: mid, ChannelID: cid, RoundID: rid}},
// 		RoundCompleted: false,
// 	}
// 	err := st.store()
// 	if err != nil {
// 		t.Fatalf("Failed to store byRound: %+v", err)
// 	}

// 	st2 := loadSendTracker(&mockClient{}, kv, nil, nil, nil, crng)
// 	if len(st2.byRound) != len(st.byRound) {
// 		t.Fatalf("byRound was not properly loaded")
// 	}
// }

// func TestRoundResult_callback(t *testing.T) {
// 	kv := versioned.NewKV(ekv.MakeMemstore())
// 	triggerCh := make(chan bool)
// 	update := func(uuid uint64, messageID cryptoMessage.ID,
// 		timestamp time.Time, round rounds.Round, status SentStatus) {
// 		triggerCh <- true
// 	}
// 	trigger := func(chID *id.ID, umi *userMessageInternal, ts time.Time,
// 		receptionID receptionID.EphemeralIdentity, round rounds.Round,
// 		status SentStatus) (uint64, error) {
// 		return 0, nil
// 	}

// 	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

// 	st := loadSendTracker(&mockClient{}, kv, trigger, nil, update, crng)

// 	cid := id.NewIdFromString("channel", id.User, t)
// 	rid := id.Round(2)
// 	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid), []byte("hello"))
// 	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
// 		userMessage: &UserMessage{},
// 		channelMessage: &ChannelMessage{
// 			Lease:       0,
// 			RoundID:     uint64(rid),
// 			PayloadType: 0,
// 			Payload:     []byte("hello"),
// 		},
// 		messageID: mid,
// 	})
// 	if err != nil {
// 		t.Fatalf(err.Error())
// 	}

// 	err = st.send(uuid, mid, rounds.Round{
// 		ID:    rid,
// 		State: 2,
// 	})

// 	rr := roundResults{
// 		round:     rid,
// 		st:        st,
// 		numChecks: 0,
// 	}

// 	rr.callback(true, false, map[id.Round]cmix.RoundResult{
// 		rid: {Status: cmix.Succeeded, Round: rounds.Round{ID: rid, State: 0}}})

// 	timeout := time.NewTicker(time.Second * 5)
// 	select {
// 	case <-triggerCh:
// 	case <-timeout.C:
// 		t.Fatal("Did not receive update")
// 	}
// }