////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	// tenMsInNs is a prime close to one million to ensure that
	// patterns do not arise due to cofactors with the message ID
	// when doing the modulo.
	tenMsInNs     = 10000019
	halfTenMsInNs = tenMsInNs / 2
)

func Test_initEvents(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// verify the model is registered
	if e.model != me {
		t.Errorf("Event model is not registered")
	}

	// check registered channels was created
	if e.registered == nil {
		t.Fatalf("Registered handlers is not registered")
	}

	// check that all the default callbacks are registered
	if len(e.registered) != 7 {
		t.Errorf("The correct number of default handlers are not "+
			"registered; %d vs %d", len(e.registered), 7)
		// If this fails, is means the default handlers have changed. edit the
		// number here and add tests below. be suspicious if it goes down.
	}

	if getFuncName(e.registered[Text].listener) != getFuncName(e.receiveTextMessage) {
		t.Errorf("Text does not have recieveTextMessageRegistred")
	}

	if getFuncName(e.registered[AdminText].listener) != getFuncName(e.receiveTextMessage) {
		t.Errorf("AdminText does not have recieveTextMessageRegistred")
	}

	if getFuncName(e.registered[Reaction].listener) != getFuncName(e.receiveReaction) {
		t.Errorf("Reaction does not have recieveReaction")
	}
}

// Unit test of NewReceiveMessageHandler.
func TestNewReceiveMessageHandler(t *testing.T) {
	expected := &ReceiveMessageHandler{
		name:       "handlerName",
		userSpace:  true,
		adminSpace: true,
		mutedSpace: true,
	}

	received := NewReceiveMessageHandler(expected.name, expected.listener,
		expected.userSpace, expected.adminSpace, expected.mutedSpace)

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("New ReceiveMessageHandler does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that ReceiveMessageHandler.CheckSpace returns the expected output for
// every possible combination of user, admin, and muted space.
func TestReceiveMessageHandler_CheckSpace(t *testing.T) {
	handlers := []struct {
		*ReceiveMessageHandler
		expected []bool
	}{
		{NewReceiveMessageHandler("0", nil, true, true, true),
			[]bool{true, true, true, true, true, true, false, false}},
		{NewReceiveMessageHandler("1", nil, true, true, false),
			[]bool{false, true, false, true, false, true, false, false}},
		{NewReceiveMessageHandler("2", nil, true, false, true),
			[]bool{true, true, true, true, false, false, false, false}},
		{NewReceiveMessageHandler("3", nil, true, false, false),
			[]bool{false, true, false, true, false, false, false, false}},
		{NewReceiveMessageHandler("4", nil, false, true, true),
			[]bool{true, true, false, false, true, true, false, false}},
		{NewReceiveMessageHandler("5", nil, false, true, false),
			[]bool{false, true, false, false, false, true, false, false}},
		{NewReceiveMessageHandler("6", nil, false, false, true),
			[]bool{false, false, false, false, false, false, false, false}},
		{NewReceiveMessageHandler("7", nil, false, false, false),
			[]bool{false, false, false, false, false, false, false, false}},
	}

	tests := []struct{ user, admin, muted bool }{
		{true, true, true},    // 0
		{true, true, false},   // 1
		{true, false, true},   // 2
		{true, false, false},  // 3
		{false, true, true},   // 4
		{false, true, false},  // 5
		{false, false, true},  // 6
		{false, false, false}, // 7
	}

	for i, handler := range handlers {
		for j, tt := range tests {
			err := handler.CheckSpace(tt.user, tt.admin, tt.muted)
			if handler.expected[j] && err != nil {
				t.Errorf("Handler %d failed test %d: %s", i, j, err)
			} else if !handler.expected[j] && err == nil {
				t.Errorf("Handler %s (#%d) did not fail test #%d when it "+
					"should have.\nhandler: %s\ntest:    %+v",
					handler.name, i, j, handler.SpaceString(), tt)
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Message Handlers                                                           //
////////////////////////////////////////////////////////////////////////////////

func Test_events_RegisterReceiveHandler(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Test that a new reception handler can be registered.
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, NewReceiveMessageHandler(
		"reaction", e.receiveReaction, true, false, true))
	if err != nil {
		t.Fatalf("Failed to register '%s' when it should be "+
			"sucesfull: %+v", mt, err)
	}

	// check that it is written
	returnedHandler, exists := e.registered[mt]
	if !exists {
		t.Fatalf("Failed to get handler '%s' after registration", mt)
	}

	// check that the correct function is written
	if getFuncName(e.receiveReaction) != getFuncName(returnedHandler.listener) {
		t.Fatalf("Failed to get correct handler for '%s' after "+
			"registration, %s vs %s", mt, getFuncName(e.receiveReaction),
			getFuncName(returnedHandler.listener))
	}

	// test that writing to the same receive handler fails
	err = e.RegisterReceiveHandler(mt, NewReceiveMessageHandler(
		"userTextMessage", e.receiveTextMessage, true, false, true))
	if err == nil {
		t.Fatalf("Failed to register '%s' when it should be "+
			"sucesfull: %+v", mt, err)
	} else if err != MessageTypeAlreadyRegistered {
		t.Fatalf("Wrong error returned when reregierting message "+
			"tyle '%s': %+v", mt, err)
	}

	// check that it is still written
	returnedHandler, exists = e.registered[mt]
	if !exists {
		t.Fatalf("Failed to get handler '%s' after second "+
			"registration", mt)
	}

	// check that the correct function is written
	if getFuncName(e.receiveReaction) != getFuncName(returnedHandler.listener) {
		t.Fatalf("Failed to get correct handler for '%s' after "+
			"second registration, %s vs %s", mt, getFuncName(e.receiveReaction),
			getFuncName(returnedHandler.listener))
	}
}

func getFuncName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

////////////////////////////////////////////////////////////////////////////////
// Message Triggers                                                           //
////////////////////////////////////////////////////////////////////////////////

type dummyMessageTypeHandler struct {
	triggered        bool
	channelID        *id.ID
	messageID        message.ID
	messageType      MessageType
	nickname         string
	content          []byte
	encryptedPayload []byte
	timestamp        time.Time
	lease            time.Duration
	round            rounds.Round
}

func (dh *dummyMessageTypeHandler) dummyMessageTypeReceiveMessage(
	channelID *id.ID, messageID message.ID, messageType MessageType,
	nickname string, content, encryptedPayload []byte, _ ed25519.PublicKey,
	_ uint32, _ uint8, timestamp, _ time.Time, lease time.Duration,
	_ id.Round, round rounds.Round, _ SentStatus, _, _ bool) uint64 {
	dh.triggered = true
	dh.channelID = channelID
	dh.messageID = messageID
	dh.messageType = messageType
	dh.nickname = nickname
	dh.content = content
	dh.encryptedPayload = encryptedPayload
	dh.timestamp = timestamp
	dh.lease = lease
	dh.round = round
	return rand.Uint64()
}

func Test_events_triggerEvents(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	dummy := &dummyMessageTypeHandler{}

	// Register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, NewReceiveMessageHandler(
		"dummy", dummy.dummyMessageTypeReceiveMessage, true, false, true))
	if err != nil {
		t.Fatalf("Error on registration, should not have happened: %+v", err)
	}

	// Craft the input for the event
	chID := &id.ID{1}
	umi, _, _ := builtTestUMI(t, mt)
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}

	// Call the trigger
	_, err = e.triggerEvent(chID, umi, nil, netTime.Now(),
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatal(err)
	}

	// Check the data is stored in the dummy
	expected := &dummyMessageTypeHandler{true, chID, umi.GetMessageID(), mt,
		umi.channelMessage.Nickname, umi.GetChannelMessage().Payload, nil,
		dummy.timestamp, time.Duration(umi.GetChannelMessage().Lease), r}
	if !reflect.DeepEqual(expected, dummy) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, dummy)
	}

	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("Incorrect timestamp.\nexpected: %s\nreceived: %s",
			r.Timestamps[states.QUEUED], dummy.timestamp)
	}
}

func Test_events_triggerEvents_noChannel(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	dummy := &dummyMessageTypeHandler{}

	// skip handler registration
	mt := MessageType(1)

	// Craft the input for the event
	chID := &id.ID{1}

	umi, _, _ := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}

	// call the trigger
	_, err := e.triggerEvent(chID, umi, nil, netTime.Now(),
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatal(err)
	}

	// check that the event was triggered
	if dummy.triggered {
		t.Errorf("The event was triggered when it is unregistered")
	}
}

func Test_events_triggerAdminEvents(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	dummy := &dummyMessageTypeHandler{}

	// Register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, NewReceiveMessageHandler(
		"dummy", dummy.dummyMessageTypeReceiveMessage, false, true, false))
	if err != nil {
		t.Fatalf("Error on registration: %+v", err)
	}

	// Craft the input for the event
	chID := &id.ID{1}
	u, _, cm := builtTestUMI(t, mt)
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.
		DeriveChannelMessageID(chID, uint64(r.ID), u.userMessage.Message)

	// Call the trigger
	_, err = e.triggerAdminEvent(chID, cm, nil, netTime.Now(), msgID,
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatal(err)
	}

	// Check the data is stored in the dummy
	expected := &dummyMessageTypeHandler{true, chID, msgID, mt, AdminUsername,
		cm.Payload, nil, dummy.timestamp, time.Duration(cm.Lease), r}
	if !reflect.DeepEqual(expected, dummy) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, dummy)
	}

	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("Incorrect timestamp.\nexpected: %s\nreceived: %s",
			r.Timestamps[states.QUEUED], dummy.timestamp)
	}
}

func Test_events_triggerAdminEvents_noChannel(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	dummy := &dummyMessageTypeHandler{}
	mt := AdminText

	// Craft the input for the event
	chID := &id.ID{1}
	u, _, cm := builtTestUMI(t, mt)
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.
		DeriveChannelMessageID(chID, uint64(r.ID), u.userMessage.Message)

	// Call the trigger
	_, err := e.triggerAdminEvent(chID, cm, nil, netTime.Now(), msgID,
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatal(err)
	}

	// Check that the event was not triggered
	if dummy.triggered {
		t.Errorf("The admin event was triggered when unregistered")
	}
}
func TestEvents_triggerActionEvent(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(&MockEvent{}, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))
	dummy := &dummyMessageTypeHandler{}

	// Register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, NewReceiveMessageHandler(
		"dummy", dummy.dummyMessageTypeReceiveMessage, false, true, false))
	if err != nil {
		t.Fatalf("Error on registration: %+v", err)
	}

	// Craft the input for the event
	chID := &id.ID{1}
	u, _, cm := builtTestUMI(t, mt)
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.
		DeriveChannelMessageID(chID, uint64(r.ID), u.userMessage.Message)

	// Call the trigger
	_, err = e.triggerActionEvent(chID, msgID, MessageType(cm.PayloadType),
		cm.Nickname, cm.Payload, nil, netTime.Now(), netTime.Now(),
		time.Duration(cm.Lease), r.ID, r, Delivered, true)
	if err != nil {
		t.Fatal(err)
	}

	// Check the data is stored in the dummy
	expected := &dummyMessageTypeHandler{true, chID, msgID, mt, cm.Nickname,
		cm.Payload, nil, dummy.timestamp, time.Duration(cm.Lease), r}
	if !reflect.DeepEqual(expected, dummy) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, dummy)
	}

	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("Incorrect timestamp.\nexpected: %s\nreceived: %s",
			r.Timestamps[states.QUEUED], dummy.timestamp)
	}
}

func Test_events_receiveTextMessage_Message(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: nil,
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}
	senderNickname := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	dmToken := uint32(8675309)
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)

	// Call the handler
	e.receiveTextMessage(chID, msgID, Text, senderNickname, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	// Check the results on the model
	expected := eventReceive{chID, msgID, message.ID{}, senderNickname,
		[]byte(textPayload.Text), ts, lease, r, Delivered, false, false, Text,
		dmToken, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}

	if !me.eventReceive.reactionTo.Equals(message.ID{}) {
		t.Errorf("Reaction ID is not blank, %s", me.eventReceive.reactionTo)
	}
}

func Test_events_receiveTextMessage_Reply(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	replyMsgId := message.DeriveChannelMessageID(chID, uint64(r.ID), []byte("blarg"))
	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: replyMsgId[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatal(err)
	}
	dmToken := uint32(8675309)

	// Call the handler
	e.receiveTextMessage(chID, msgID, Text, senderUsername, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	// Check the results on the model
	expected := eventReceive{chID, msgID, replyMsgId,
		senderUsername, []byte(textPayload.Text), ts, lease, r, Delivered,
		false, false, Text, dmToken, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

func Test_events_receiveTextMessage_Reply_BadReply(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	replyMsgId := []byte("blarg")
	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: replyMsgId[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatal(err)
	}
	dmToken := uint32(8675309)

	// Call the handler
	e.receiveTextMessage(chID, msgID, Text, senderUsername, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	// Check the results on the model
	expected := eventReceive{chID, msgID, message.ID{},
		senderUsername, []byte(textPayload.Text), ts, lease, r, Delivered,
		false, false, Text, dmToken, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

func Test_events_receiveReaction(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	replyMsgId := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "🍆",
		ReactionMessageID: replyMsgId[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	msgID := message.DeriveChannelMessageID(chID, 420, textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatal(err)
	}
	dmToken := uint32(8675309)

	// Call the handler
	e.receiveReaction(chID, msgID, Reaction, senderUsername, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	// Check the results on the model
	expected := eventReceive{chID, msgID, replyMsgId, senderUsername,
		[]byte(textPayload.Reaction), ts, lease, r, Delivered, false, false,
		Reaction, dmToken, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

func Test_events_receiveReaction_InvalidReactionMessageID(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	replyMsgId := []byte("blarg")
	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "🍆",
		ReactionMessageID: replyMsgId[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	msgID := message.DeriveChannelMessageID(chID, 420, textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatal(err)
	}
	dmToken := uint32(8675309)

	// Call the handler
	e.receiveReaction(chID, msgID, Reaction, senderUsername, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, 0, r.ID, r, Delivered,
		false, false)

	// Check the results on the model
	expected := eventReceive{nil, message.ID{}, message.ID{}, "", nil,
		time.Time{}, 0, rounds.Round{}, 0, false, false, 0, 0, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

func Test_events_receiveReaction_InvalidReactionContent(t *testing.T) {
	me := &MockEvent{}
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID := &id.ID{1}
	replyMsgId := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "I'm not a reaction",
		ReactionMessageID: replyMsgId[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshal the message proto: %+v", err)
	}
	msgID := message.DeriveChannelMessageID(chID, 420, textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	pi, err := cryptoChannel.GenerateIdentity(rand.New(rand.NewSource(64)))
	if err != nil {
		t.Fatal(err)
	}
	dmToken := uint32(8675309)

	// Call the handler
	e.receiveReaction(chID, msgID, Reaction, senderUsername, textMarshaled, nil,
		pi.PubKey, dmToken, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	// Check the results on the model
	expected := eventReceive{nil, message.ID{}, message.ID{}, "", nil,
		time.Time{}, 0, rounds.Round{}, 0, false, false, 0, 0, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

// Unit test of events.receiveDelete.
func Test_events_receiveDelete(t *testing.T) {
	me, prng := &MockEvent{}, rand.New(rand.NewSource(65))
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID, _ := id.NewRandomID(prng, id.User)
	targetMessageID := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	textPayload := &CMIXChannelDelete{
		Version:   0,
		MessageID: targetMessageID[:],
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to proto marshal %T: %+v", textPayload, err)
	}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatal(err)
	}

	me.eventReceive = eventReceive{chID, message.ID{},
		targetMessageID, senderUsername, textMarshaled, ts, lease, r, Delivered,
		false, false, Text, 0, 0}

	// Call the handler
	e.receiveDelete(chID, msgID, Delete, AdminUsername, textMarshaled, nil,
		pi.PubKey, 0, pi.CodesetVersion, ts, ts, lease, r.ID, r, Delivered,
		true, false)

	// Check the results on the model
	if !reflect.DeepEqual(me.eventReceive, eventReceive{}) {
		t.Errorf("Message not deleted.\nexpected: %v\nreceived: %v",
			eventReceive{}, me.eventReceive)
	}
}

// Unit test of events.receivePinned.
func Test_events_receivePinned(t *testing.T) {
	me, prng := &MockEvent{}, rand.New(rand.NewSource(65))
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID, _ := id.NewRandomID(prng, id.User)
	targetMessageID := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	textPayload := &CMIXChannelPinned{
		Version:    0,
		MessageID:  targetMessageID[:],
		UndoAction: false,
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to proto marshal %T: %+v", textPayload, err)
	}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("someTest")
	me.eventReceive = eventReceive{chID, message.ID{},
		targetMessageID, senderUsername, content, ts, lease, r, Delivered,
		false, false, Text, 0, 0}

	// Call the handler
	e.receivePinned(chID, msgID, Pinned, senderUsername, textMarshaled, nil,
		pi.PubKey, 0, pi.CodesetVersion, ts, ts, lease, r.ID, r, Delivered,
		true, false)

	// Check the results on the model
	expected := eventReceive{chID, message.ID{}, targetMessageID,
		senderUsername, content, ts, lease, r, Delivered, true, false, Text, 0, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

// Unit test of events.receivePinned.
func Test_events_receiveMute(t *testing.T) {
	me, prng := &MockEvent{}, rand.New(rand.NewSource(65))
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID, _ := id.NewRandomID(prng, id.User)
	targetMessageID := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	pubKey, _, _ := ed25519.GenerateKey(prng)
	textPayload := &CMIXChannelMute{
		Version:    0,
		PubKey:     pubKey,
		UndoAction: false,
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to proto marshal %T: %+v", textPayload, err)
	}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("someTest")
	me.eventReceive = eventReceive{chID, message.ID{}, targetMessageID,
		senderUsername, content, ts, lease, r, Delivered, false, false, Text, 0,
		0}

	// Call the handler
	e.receiveMute(chID, msgID, Mute, senderUsername, textMarshaled, nil,
		pi.PubKey, 0, pi.CodesetVersion, ts, ts, lease, r.ID, r, Delivered,
		true, false)

	// Check the results on the model
	expected := eventReceive{chID, message.ID{},
		targetMessageID, senderUsername, content, ts, lease, r, Delivered,
		false, false, Text, 0, 0}
	if !reflect.DeepEqual(expected, me.eventReceive) {
		t.Errorf("Did not receive expected values."+
			"\nexpected: %+v\nreceived: %+v", expected, me.eventReceive)
	}
}

// Unit test of events.receiveAdminReplay.
func Test_events_receiveAdminReplay(t *testing.T) {
	me, prng := &MockEvent{}, csprng.NewSystemRNG()
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	e := initEvents(me, 512, kv,
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG))

	// Craft the input for the event
	chID, _ := id.NewRandomID(prng, id.User)
	targetMessageID := message.DeriveChannelMessageID(chID, 420, []byte("blarg"))
	textPayload := &CMIXChannelPinned{
		Version:    0,
		MessageID:  targetMessageID[:],
		UndoAction: false,
	}
	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to proto marshal %T: %+v", textPayload, err)
	}

	ch, pk, err := newTestChannel("abc", "", prng, cryptoBroadcast.Public)
	if err != nil {
		t.Fatalf("Failed to generate channel: %+v", err)
	}
	cipherText, _, _, _, err :=
		ch.EncryptRSAToPublic(textMarshaled, pk, 3072, prng)
	if err != nil {
		t.Fatalf("Failed to encrypt RSAToPublic: %+v", err)
	}
	r := rounds.Round{ID: 420,
		Timestamps: map[states.Round]time.Time{states.QUEUED: netTime.Now()}}
	msgID := message.DeriveChannelMessageID(chID, uint64(r.ID), textMarshaled)
	senderUsername := "Alice"
	ts := netTime.Now()
	lease := 69 * time.Minute
	pi, err := cryptoChannel.GenerateIdentity(prng)
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("someTest")
	me.eventReceive = eventReceive{chID, message.ID{},
		targetMessageID, senderUsername, content, ts, lease, r, Delivered,
		false, false, Text, 0, 0}

	c := make(chan []byte)
	e.broadcast.addProcessor(
		chID, adminProcessor, &testAdminProcessor{adminMsgChan: c})

	// Call the handler
	e.receiveAdminReplay(chID, msgID, AdminReplay, senderUsername, cipherText,
		nil, pi.PubKey, 0, pi.CodesetVersion, ts, ts, lease, r.ID, r,
		Delivered, false, false)

	select {
	case encrypted := <-c:
		decrypted, err2 := ch.DecryptRSAToPublicInner(encrypted)
		if err2 != nil {
			t.Errorf("Failed to decrypt message: %+v", err2)
		}

		received := &CMIXChannelPinned{}
		err = proto.Unmarshal(decrypted, received)
		if err != nil {
			t.Errorf("Failed to proto unmarshal message: %+v", err)
		}

		if !proto.Equal(textPayload, received) {
			t.Errorf("Received admin message does not match expected."+
				"\nexpected: %s\nreceived: %s", textPayload, received)
		}

	case <-time.After(15 * time.Millisecond):
		t.Errorf("Timed out waiting for processor to be called.")
	}
}

// //////////////////////////////////////////////////////////////////////////////
// Mock Event Model                                                           //
// //////////////////////////////////////////////////////////////////////////////
type eventReceive struct {
	channelID   *id.ID
	messageID   message.ID
	reactionTo  message.ID
	nickname    string
	content     []byte
	timestamp   time.Time
	lease       time.Duration
	round       rounds.Round
	status      SentStatus
	pinned      bool
	hidden      bool
	messageType MessageType
	dmToken     uint32
	codeset     uint8
}

// Verify that MockEvent adheres to the EventModel interface.
var _ EventModel = (*MockEvent)(nil)

// MockEvents adheres to the EventModel interface and is used for testing.
type MockEvent struct {
	uuid uint64
	eventReceive
}

func (m *MockEvent) getUUID() uint64 {
	old := m.uuid
	m.uuid++
	return old
}

func (*MockEvent) JoinChannel(*cryptoBroadcast.Channel) {}
func (*MockEvent) LeaveChannel(*id.ID)                  {}
func (m *MockEvent) ReceiveMessage(channelID *id.ID,
	messageID message.ID, nickname, text string,
	_ ed25519.PublicKey, dmToken uint32, codeset uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, messageType MessageType,
	status SentStatus, hidden bool) uint64 {
	m.eventReceive = eventReceive{
		channelID:   channelID,
		messageID:   messageID,
		reactionTo:  message.ID{},
		nickname:    nickname,
		content:     []byte(text),
		timestamp:   timestamp,
		lease:       lease,
		round:       round,
		status:      status,
		pinned:      false,
		hidden:      hidden,
		messageType: messageType,
		dmToken:     dmToken,
		codeset:     codeset,
	}
	return m.getUUID()
}
func (m *MockEvent) ReceiveReply(channelID *id.ID, messageID,
	reactionTo message.ID, nickname, text string,
	_ ed25519.PublicKey, dmToken uint32, codeset uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, messageType MessageType,
	status SentStatus, hidden bool) uint64 {
	m.eventReceive = eventReceive{
		channelID:   channelID,
		messageID:   messageID,
		reactionTo:  reactionTo,
		nickname:    nickname,
		content:     []byte(text),
		timestamp:   timestamp,
		lease:       lease,
		round:       round,
		status:      status,
		pinned:      false,
		hidden:      hidden,
		messageType: messageType,
		dmToken:     dmToken,
		codeset:     codeset,
	}
	return m.getUUID()
}
func (m *MockEvent) ReceiveReaction(channelID *id.ID, messageID,
	reactionTo message.ID, nickname, reaction string,
	_ ed25519.PublicKey, dmToken uint32, codeset uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, messageType MessageType,
	status SentStatus, hidden bool) uint64 {
	m.eventReceive = eventReceive{
		channelID:   channelID,
		messageID:   messageID,
		reactionTo:  reactionTo,
		nickname:    nickname,
		content:     []byte(reaction),
		timestamp:   timestamp,
		lease:       lease,
		round:       round,
		status:      status,
		pinned:      false,
		hidden:      hidden,
		messageType: messageType,
		dmToken:     dmToken,
		codeset:     codeset,
	}
	return m.getUUID()
}

func (m *MockEvent) UpdateFromUUID(_ uint64, messageID *message.ID,
	timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
	status *SentStatus) {

	if messageID != nil {
		m.eventReceive.messageID = *messageID
	}
	if timestamp != nil {
		m.eventReceive.timestamp = *timestamp
	}
	if round != nil {
		m.eventReceive.round = *round
	}
	if status != nil {
		m.eventReceive.status = *status
	}
	if pinned != nil {
		m.eventReceive.pinned = *pinned
	}
	if hidden != nil {
		m.eventReceive.hidden = *hidden
	}
}

func (m *MockEvent) UpdateFromMessageID(_ message.ID,
	timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
	status *SentStatus) uint64 {

	if timestamp != nil {
		m.eventReceive.timestamp = *timestamp
	}
	if round != nil {
		m.eventReceive.round = *round
	}
	if status != nil {
		m.eventReceive.status = *status
	}
	if pinned != nil {
		m.eventReceive.pinned = *pinned
	}
	if hidden != nil {
		m.eventReceive.hidden = *hidden
	}

	return m.getUUID()
}

func (m *MockEvent) GetMessage(message.ID) (ModelMessage, error) {
	return ModelMessage{
		UUID:            m.getUUID(),
		Nickname:        m.eventReceive.nickname,
		MessageID:       m.eventReceive.messageID,
		ChannelID:       m.eventReceive.channelID,
		ParentMessageID: m.reactionTo,
		Timestamp:       m.eventReceive.timestamp,
		Lease:           m.eventReceive.lease,
		Status:          m.status,
		Hidden:          m.hidden,
		Pinned:          m.pinned,
		Content:         m.eventReceive.content,
		Type:            m.messageType,
		Round:           m.round.ID,
		PubKey:          nil,
		CodesetVersion:  m.codeset,
	}, nil
}

func (m *MockEvent) MuteUser(channelID *id.ID, pubKey ed25519.PublicKey, unmute bool) {
	return
}

func (m *MockEvent) DeleteMessage(message.ID) error {
	m.eventReceive = eventReceive{}
	return nil
}

// withinMutationWindow is a utility test function to check if a mutated
// timestamp is within the allowable window
func withinMutationWindow(raw, mutated time.Time) bool {
	lowerBound := raw.Add(-time.Duration(halfTenMsInNs))
	upperBound := raw.Add(time.Duration(halfTenMsInNs))

	return mutated.After(lowerBound) && mutated.Before(upperBound)
}
