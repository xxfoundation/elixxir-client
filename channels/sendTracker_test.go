package channels

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"testing"
	"time"
)

type mockClient struct{}

func (mc *mockClient) GetMaxMessageLength() int {
	return 2048
}
func (mc *mockClient) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	return rounds.Round{}, ephemeral.Id{}, nil
}
func (mc *mockClient) IsHealthy() bool {
	return true
}
func (mc *mockClient) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {}
func (mc *mockClient) AddService(clientID *id.ID, newService message.Service,
	response message.Processor) {
}
func (mc *mockClient) DeleteClientService(clientID *id.ID) {}
func (mc *mockClient) RemoveIdentity(id *id.ID)            {}
func (mc *mockClient) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback,
	roundList ...id.Round) {
}
func (mc *mockClient) AddHealthCallback(f func(bool)) uint64 {
	return 0
}
func (mc *mockClient) RemoveHealthCallback(uint64) {}

// Test MessageReceive basic logic
func TestSendTracker_MessageReceive(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	trigger := func(chID *id.ID, umi *userMessageInternal,
		receptionID receptionID.EphemeralIdentity, round rounds.Round, status SentStatus) {
	}

	st := loadSendTracker(&mockClient{}, kv, trigger, nil, nil)

	mid := cryptoChannel.MakeMessageID([]byte("hello"))
	process := st.MessageReceive(mid)
	if process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}

	cid := id.NewIdFromString("channel", id.User, t)
	rid := id.Round(2)
	st.send(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:       time.Now().UnixNano(),
			RoundID:     uint64(rid),
			PayloadType: 0,
			Payload:     []byte("hello"),
		},
		messageID: mid,
	}, rounds.Round{
		ID:    rid,
		State: 1,
	})
	process = st.MessageReceive(mid)
	if !process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}

	cid2 := id.NewIdFromString("channel two", id.User, t)
	st.send(cid2, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:       time.Now().UnixNano(),
			RoundID:     uint64(rid),
			PayloadType: 0,
			Payload:     []byte("hello again"),
		},
		messageID: mid,
	}, rounds.Round{
		ID:    rid,
		State: 1,
	})
	process = st.MessageReceive(mid)
	if !process {
		t.Fatalf("Did not receive expected result from MessageReceive")
	}
}

// Test sendAdmin function, confirming that data is stored appropriately
// and callbacks are called
func TestSendTracker_sendAdmin(t *testing.T) {
	triggerCh := make(chan bool)

	kv := versioned.NewKV(ekv.MakeMemstore())

	adminTrigger := func(chID *id.ID, cm *ChannelMessage,
		messageID cryptoChannel.MessageID, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, status SentStatus) {
		triggerCh <- true
	}

	st := loadSendTracker(&mockClient{}, kv, nil, adminTrigger, nil)

	cid := id.NewIdFromString("channel", id.User, t)
	mid := cryptoChannel.MakeMessageID([]byte("hello"))
	rid := id.Round(2)
	st.sendAdmin(cid, &ChannelMessage{
		Lease:       0,
		RoundID:     uint64(rid),
		PayloadType: 0,
		Payload:     []byte("hello"),
	}, mid, rounds.Round{
		ID:    rid,
		State: 2,
	})

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
		t.Log("Received over trigger chan")
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	if !ok {
		t.Fatal("Should have found a tracked round")
	}
	if len(trackedRound) != 1 {
		t.Fatal("Did not find expected number of trackedRounds")
	}
	if trackedRound[0].MsgID != mid {
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

// Test send tracker send function, confirming that data is stored appropriately
//// and callbacks are called
func TestSendTracker_send(t *testing.T) {
	triggerCh := make(chan bool)

	kv := versioned.NewKV(ekv.MakeMemstore())
	trigger := func(chID *id.ID, umi *userMessageInternal,
		receptionID receptionID.EphemeralIdentity, round rounds.Round, status SentStatus) {
		triggerCh <- true
	}

	st := loadSendTracker(&mockClient{}, kv, trigger, nil, nil)

	cid := id.NewIdFromString("channel", id.User, t)
	mid := cryptoChannel.MakeMessageID([]byte("hello"))
	rid := id.Round(2)
	st.send(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:       0,
			RoundID:     uint64(rid),
			PayloadType: 0,
			Payload:     []byte("hello"),
		},
		messageID: mid,
	}, rounds.Round{
		ID:    rid,
		State: 2,
	})

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
		t.Log("Received over trigger chan")
	case <-timeout.C:
		t.Fatal("Timed out waiting for trigger chan")
	}

	trackedRound, ok := st.byRound[rid]
	if !ok {
		t.Fatal("Should have found a tracked round")
	}
	if len(trackedRound) != 1 {
		t.Fatal("Did not find expected number of trackedRounds")
	}
	if trackedRound[0].MsgID != mid {
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

// Test loading stored byRound map from storage
func TestSendTracker_load_store(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())

	st := loadSendTracker(&mockClient{}, kv, nil, nil, nil)
	cid := id.NewIdFromString("channel", id.User, t)
	mid := cryptoChannel.MakeMessageID([]byte("hello"))
	rid := id.Round(2)
	st.byRound[rid] = []*tracked{{MsgID: mid, ChannelID: cid, RoundID: rid}}
	err := st.store()
	if err != nil {
		t.Fatalf("Failed to store byRound: %+v", err)
	}

	st2 := loadSendTracker(&mockClient{}, kv, nil, nil, nil)
	if len(st2.byRound) != len(st.byRound) {
		t.Fatalf("byRound was not properly loaded")
	}
}

func TestRoundResult_callback(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	triggerCh := make(chan bool)
	update := func(messageID cryptoChannel.MessageID, status SentStatus) {
		triggerCh <- true
	}
	trigger := func(chID *id.ID, umi *userMessageInternal,
		receptionID receptionID.EphemeralIdentity, round rounds.Round, status SentStatus) {
	}
	st := loadSendTracker(&mockClient{}, kv, trigger, nil, update)

	cid := id.NewIdFromString("channel", id.User, t)
	mid := cryptoChannel.MakeMessageID([]byte("hello"))
	rid := id.Round(2)
	st.send(cid, &userMessageInternal{
		userMessage: &UserMessage{},
		channelMessage: &ChannelMessage{
			Lease:       0,
			RoundID:     uint64(rid),
			PayloadType: 0,
			Payload:     []byte("hello"),
		},
		messageID: mid,
	}, rounds.Round{
		ID:    rid,
		State: 2,
	})

	rr := roundResults{
		round:     rid,
		st:        st,
		numChecks: 0,
	}

	rr.callback(true, false, map[id.Round]cmix.RoundResult{rid: {cmix.Succeeded, rounds.Round{
		ID:    rid,
		State: 0,
	}}})

	timeout := time.NewTicker(time.Second * 5)
	select {
	case <-triggerCh:
		t.Log("Received trigger")
	case <-timeout.C:
		t.Fatal("Did not receive update")
	}
}
