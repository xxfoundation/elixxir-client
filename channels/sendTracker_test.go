package channels

import (
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

type mockClient struct{}

func (mc *mockClient) GetMaxMessageLength() int { return 2048 }
func (mc *mockClient) SendWithAssembler(*id.ID, cmix.MessageAssembler,
	cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	return rounds.Round{}, ephemeral.Id{}, nil
}
func (mc *mockClient) AddIdentity(*id.ID, time.Time, bool, message.Processor)                       {}
func (mc *mockClient) AddIdentityWithHistory(*id.ID, time.Time, time.Time, bool, message.Processor) {}
func (mc *mockClient) RemoveIdentity(*id.ID)                                                        {}
func (mc *mockClient) AddService(*id.ID, message.Service, message.Processor)                        {}
func (mc *mockClient) UpsertCompressedService(*id.ID, message.CompressedService, message.Processor) {}
func (mc *mockClient) DeleteClientService(*id.ID)                                                   {}
func (mc *mockClient) IsHealthy() bool                                                              { return true }
func (mc *mockClient) AddHealthCallback(func(bool)) uint64                                          { return 0 }
func (mc *mockClient) RemoveHealthCallback(uint64)                                                  {}
func (mc *mockClient) GetRoundResults(time.Duration, cmix.RoundEventCallback, ...id.Round)          {}

// Test MessageReceive basic logic.
func TestSendTracker_MessageReceive(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	uuidNum := uint64(0)
	rid := id.Round(2)

	r := rounds.Round{
		ID:         rid,
		Timestamps: make(map[states.Round]time.Time),
	}
	r.Timestamps[states.QUEUED] = netTime.Now()
	trigger := func(*id.ID, *userMessageInternal, []byte, time.Time,
		receptionID.EphemeralIdentity, rounds.Round, SentStatus) (uint64, error) {
		oldUUID := uuidNum
		uuidNum++
		return oldUUID, nil
	}

	updateStatus := func(uint64, *cryptoMessage.ID, *time.Time, *rounds.Round,
		*bool, *bool, *SentStatus) error {
		return nil
	}

	cid := id.NewIdFromString("channel", id.User, t)

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	st := loadSendTracker(&mockClient{}, kv, trigger, nil, updateStatus, crng)

	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
		[]byte("hello"))
	process := st.MessageReceive(mid, r)
	if process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}

	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:   netTime.Now().UnixNano(),
			RoundID: uint64(rid),
			Payload: []byte("hello"),
		}}, 42)
	if err != nil {
		t.Fatal(err)
	}

	err = st.send(uuid, mid, rounds.Round{
		ID:    rid,
		State: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	process = st.MessageReceive(mid, r)
	if !process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}

	cid2 := id.NewIdFromString("channel two", id.User, t)
	uuid2, err := st.denotePendingSend(cid2, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:   netTime.Now().UnixNano(),
			RoundID: uint64(rid),
			Payload: []byte("hello again"),
		}}, 42)
	if err != nil {
		t.Fatal(err)
	}

	err = st.send(uuid2, mid, rounds.Round{
		ID:    rid,
		State: 1,
	})
	process = st.MessageReceive(mid, r)
	if !process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}
}

// Test failedSend function, confirming that data is stored appropriately and
// callbacks are called.
func TestSendTracker_failedSend(t *testing.T) {
	triggerCh := make(chan SentStatus)

	kv := versioned.NewKV(ekv.MakeMemstore())

	adminTrigger := func(*id.ID, *ChannelMessage, MessageType, []byte, time.Time,
		cryptoMessage.ID, receptionID.EphemeralIdentity, rounds.Round,
		SentStatus) (uint64, error) {
		return 0, nil
	}

	updateStatus := func(_ uint64, _ *cryptoMessage.ID, _ *time.Time,
		_ *rounds.Round, _ *bool, _ *bool, status *SentStatus) error {
		triggerCh <- *status
		return nil
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	st := loadSendTracker(&mockClient{}, kv, nil, adminTrigger, updateStatus, crng)

	cid := id.NewIdFromString("channel", id.User, t)
	rid := id.Round(2)
	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
		[]byte("hello"))
	cm := &ChannelMessage{
		Lease:   0,
		RoundID: uint64(rid),
		Payload: []byte("hello"),
	}
	uuid, err := st.denotePendingAdminSend(cid, cm, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = st.failedSend(uuid)
	if err != nil {
		t.Fatal(err)
	}

	timeout := time.NewTicker(time.Second * 5)
	select {
	case s := <-triggerCh:
		if s != Failed {
			t.Fatalf("Did not receive failed from failed message")
		}
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	if ok {
		t.Fatal("Should not have found a tracked round")
	}
	if len(trackedRound.List) != 0 {
		t.Fatal("Did not find expected number of trackedRounds")
	}

	_, ok = st.byMessageID[mid]
	if ok {
		t.Error("Should not have found tracked message")
	}

	_, ok = st.unsent[uuid]
	if ok {
		t.Fatal("Should not have found an unsent")
	}
}

// Test send tracker send function, confirming that data is stored appropriately
// and callbacks are called
func TestSendTracker_send(t *testing.T) {
	triggerCh := make(chan bool)

	kv := versioned.NewKV(ekv.MakeMemstore())
	trigger := func(*id.ID, *userMessageInternal, []byte, time.Time,
		receptionID.EphemeralIdentity, rounds.Round, SentStatus) (uint64, error) {
		return 0, nil
	}

	updateStatus := func(uint64, *cryptoMessage.ID, *time.Time, *rounds.Round,
		*bool, *bool, *SentStatus) error {
		triggerCh <- true
		return nil
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	st := loadSendTracker(&mockClient{}, kv, trigger, nil, updateStatus, crng)

	cid := id.NewIdFromString("channel", id.User, t)
	rid := id.Round(2)
	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
		[]byte("hello"))
	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:   0,
			RoundID: uint64(rid),
			Payload: []byte("hello"),
		},
		messageID: mid,
	}, 42)
	if err != nil {
		t.Fatal(err)
	}

	err = st.send(uuid, mid, rounds.Round{
		ID:    rid,
		State: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	if !ok {
		t.Fatal("Should have found a tracked round")
	}
	if len(trackedRound.List) != 1 {
		t.Fatal("Did not find expected number of trackedRounds")
	}
	if trackedRound.List[0].MsgID != mid {
		t.Fatalf("Did not find expected message ID in trackedRounds")
	}

	trackedMsg, ok := st.byMessageID[mid]
	if !ok {
		t.Error("Should have found tracked message")
	}
	if trackedMsg.MsgID != mid {
		t.Fatalf("Did not find expected message ID in byMessageID")
	}
}

// Test loading stored byRound map from storage.
func TestSendTracker_load_store(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	st := loadSendTracker(&mockClient{}, kv, nil, nil, nil, crng)
	cid := id.NewIdFromString("channel", id.User, t)
	rid := id.Round(2)
	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid),
		[]byte("hello"))
	st.byRound[rid] = trackedList{
		List:           []*tracked{{MsgID: mid, ChannelID: cid, RoundID: rid}},
		RoundCompleted: false,
	}
	err := st.store()
	if err != nil {
		t.Fatalf("Failed to store byRound: %+v", err)
	}

	st2 := loadSendTracker(&mockClient{}, kv, nil, nil, nil, crng)
	if len(st2.byRound) != len(st.byRound) {
		t.Fatalf("byRound was not properly loaded")
	}
}

func TestRoundResult_callback(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	triggerCh := make(chan bool)
	update := func(uint64, *cryptoMessage.ID, *time.Time, *rounds.Round, *bool,
		*bool, *SentStatus) error {
		triggerCh <- true
		return nil
	}
	trigger := func(*id.ID, *userMessageInternal, []byte, time.Time,
		receptionID.EphemeralIdentity, rounds.Round, SentStatus) (uint64, error) {
		return 0, nil
	}

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)

	st := loadSendTracker(&mockClient{}, kv, trigger, nil, update, crng)

	cid := id.NewIdFromString("channel", id.User, t)
	rid := id.Round(2)
	mid := cryptoMessage.DeriveChannelMessageID(cid, uint64(rid), []byte("hello"))
	uuid, err := st.denotePendingSend(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:   0,
			RoundID: uint64(rid),
			Payload: []byte("hello"),
		},
		messageID: mid,
	}, 42)
	if err != nil {
		t.Fatal(err)
	}

	err = st.send(uuid, mid, rounds.Round{
		ID:    rid,
		State: 2,
	})

	rr := roundResults{
		round:     rid,
		st:        st,
		numChecks: 0,
	}

	rr.callback(true, false, map[id.Round]cmix.RoundResult{
		rid: {Status: cmix.Succeeded, Round: rounds.Round{ID: rid, State: 0}}})

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
	case <-timeout.C:
		t.Fatal("Did not receive update")
	}
}
