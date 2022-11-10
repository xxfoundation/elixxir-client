////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"bytes"
	"crypto/ed25519"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"runtime"
	"testing"
	"time"
)

type eventReceive struct {
	channelID  *id.ID
	messageID  cryptoChannel.MessageID
	reactionTo cryptoChannel.MessageID
	nickname   string
	content    []byte
	timestamp  time.Time
	lease      time.Duration
	round      rounds.Round
}

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
	messageID cryptoChannel.MessageID, nickname, text string,
	_ ed25519.PublicKey, _ uint8, timestamp time.Time, lease time.Duration,
	round rounds.Round, _ MessageType, _ SentStatus) uint64 {
	m.eventReceive = eventReceive{
		channelID:  channelID,
		messageID:  messageID,
		reactionTo: cryptoChannel.MessageID{},
		nickname:   nickname,
		content:    []byte(text),
		timestamp:  timestamp,
		lease:      lease,
		round:      round,
	}
	return m.getUUID()
}
func (m *MockEvent) ReceiveReply(channelID *id.ID,
	messageID cryptoChannel.MessageID, reactionTo cryptoChannel.MessageID,
	nickname, text string, _ ed25519.PublicKey, _ uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, _ MessageType, _ SentStatus) uint64 {
	m.eventReceive = eventReceive{
		channelID:  channelID,
		messageID:  messageID,
		reactionTo: reactionTo,
		nickname:   nickname,
		content:    []byte(text),
		timestamp:  timestamp,
		lease:      lease,
		round:      round,
	}
	return m.getUUID()
}
func (m *MockEvent) ReceiveReaction(channelID *id.ID,
	messageID cryptoChannel.MessageID, reactionTo cryptoChannel.MessageID,
	nickname, reaction string, _ ed25519.PublicKey, _ uint8, timestamp time.Time,
	lease time.Duration, round rounds.Round, _ MessageType, _ SentStatus) uint64 {
	m.eventReceive = eventReceive{
		channelID:  channelID,
		messageID:  messageID,
		reactionTo: reactionTo,
		nickname:   nickname,
		content:    []byte(reaction),
		timestamp:  timestamp,
		lease:      lease,
		round:      round,
	}
	return m.getUUID()
}

func (m *MockEvent) UpdateFromUUID(uint64, *cryptoChannel.MessageID,
	*time.Time, *rounds.Round, *bool, *bool, *SentStatus) {
	panic("implement me")
}

func (m *MockEvent) UpdateFromMessageID(cryptoChannel.MessageID, *time.Time,
	*rounds.Round, *bool, *bool, *SentStatus) uint64 {
	panic("implement me")
}

func (m *MockEvent) GetMessage(cryptoChannel.MessageID) (ModelMessage, error) {
	panic("implement me")
}

func Test_initEvents(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// verify the model is registered
	if e.model != me {
		t.Errorf("Event model is not registered")
	}

	// check registered channels was created
	if e.registered == nil {
		t.Fatalf("Registered handlers is not registered")
	}

	// check that all the default callbacks are registered
	if len(e.registered) != 6 {
		t.Errorf("The correct number of default handlers are not "+
			"registered; %d vs %d", len(e.registered), 6)
		// If this fails, is means the default handlers have changed. edit the
		// number here and add tests below. be suspicious if it goes down.
	}

	if getFuncName(e.registered[Text]) != getFuncName(e.receiveTextMessage) {
		t.Errorf("Text does not have recieveTextMessageRegistred")
	}

	if getFuncName(e.registered[AdminText]) != getFuncName(e.receiveTextMessage) {
		t.Errorf("AdminText does not have recieveTextMessageRegistred")
	}

	if getFuncName(e.registered[Reaction]) != getFuncName(e.receiveReaction) {
		t.Errorf("Reaction does not have recieveReaction")
	}
}

func TestEvents_RegisterReceiveHandler(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// Test that a new reception handler can be registered.
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, e.receiveReaction)
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
	if getFuncName(e.receiveReaction) != getFuncName(returnedHandler) {
		t.Fatalf("Failed to get correct handler for '%s' after "+
			"registration, %s vs %s", mt, getFuncName(e.receiveReaction),
			getFuncName(returnedHandler))
	}

	// test that writing to the same receive handler fails
	err = e.RegisterReceiveHandler(mt, e.receiveTextMessage)
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
	if getFuncName(e.receiveReaction) != getFuncName(returnedHandler) {
		t.Fatalf("Failed to get correct handler for '%s' after "+
			"second registration, %s vs %s", mt, getFuncName(e.receiveReaction),
			getFuncName(returnedHandler))
	}
}

type dummyMessageTypeHandler struct {
	triggered   bool
	channelID   *id.ID
	messageID   cryptoChannel.MessageID
	messageType MessageType
	nickname    string
	content     []byte
	timestamp   time.Time
	lease       time.Duration
	round       rounds.Round
}

func (dmth *dummyMessageTypeHandler) dummyMessageTypeReceiveMessage(
	channelID *id.ID, messageID cryptoChannel.MessageID,
	messageType MessageType, nickname string, content []byte,
	_ ed25519.PublicKey, _ uint8, timestamp time.Time, lease time.Duration,
	round rounds.Round, _ SentStatus, _ bool) uint64 {
	dmth.triggered = true
	dmth.channelID = channelID
	dmth.messageID = messageID
	dmth.messageType = messageType
	dmth.nickname = nickname
	dmth.content = content
	dmth.timestamp = timestamp
	dmth.lease = lease
	dmth.round = round
	return rand.Uint64()
}

func TestEvents_triggerEvents(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	dummy := &dummyMessageTypeHandler{}

	// register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, dummy.dummyMessageTypeReceiveMessage)
	if err != nil {
		t.Fatalf("Error on registration, should not have happened: "+
			"%+v", err)
	}

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	umi, _, _ := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	// call the trigger
	_, err = e.triggerEvent(
		chID, umi, netTime.Now(), receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatalf(err.Error())
	}
	// check that the event was triggered
	if !dummy.triggered {
		t.Errorf("The event was not triggered")
	}

	// check the data is stored in the dummy
	if !dummy.channelID.Cmp(chID) {
		t.Errorf("The channel IDs do not match %s vs %s",
			dummy.channelID, chID)
	}

	if !dummy.messageID.Equals(umi.GetMessageID()) {
		t.Errorf("The message IDs do not match %s vs %s",
			dummy.messageID, umi.GetMessageID())
	}

	if dummy.messageType != mt {
		t.Errorf("The message types do not match %s vs %s",
			dummy.messageType, mt)
	}

	if dummy.nickname != umi.channelMessage.Nickname {
		t.Errorf("The usernames do not match %s vs %s",
			dummy.nickname, umi.channelMessage.Nickname)
	}

	if !bytes.Equal(dummy.content, umi.GetChannelMessage().Payload) {
		t.Errorf("The payloads do not match %s vs %s",
			dummy.content, umi.GetChannelMessage().Payload)
	}

	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("The timestamps do not match %s vs %s",
			dummy.timestamp, r.Timestamps[states.QUEUED])
	}

	if dummy.lease != time.Duration(umi.GetChannelMessage().Lease) {
		t.Errorf("The messge lease durations do not match %s vs %s",
			dummy.lease, time.Duration(umi.GetChannelMessage().Lease))
	}

	if dummy.round.ID != r.ID {
		t.Errorf("The messge round does not match %s vs %s",
			dummy.round.ID, r.ID)
	}
}

func TestEvents_triggerEvents_noChannel(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	dummy := &dummyMessageTypeHandler{}

	// skip handler registration
	mt := MessageType(1)

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	umi, _, _ := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	// call the trigger
	_, err := e.triggerEvent(
		chID, umi, netTime.Now(), receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check that the event was triggered
	if dummy.triggered {
		t.Errorf("The event was triggered when it is unregistered")
	}
}

func TestEvents_triggerAdminEvents(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	dummy := &dummyMessageTypeHandler{}

	// register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, dummy.dummyMessageTypeReceiveMessage)
	if err != nil {
		t.Fatalf("Error on registration, should not have happened: "+
			"%+v", err)
	}

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	u, _, cm := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	msgID := cryptoChannel.MakeMessageID(u.userMessage.Message, chID)

	// call the trigger
	_, err = e.triggerAdminEvent(chID, cm, netTime.Now(), msgID,
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check that the event was triggered
	if !dummy.triggered {
		t.Errorf("The admin event was not triggered")
	}

	// check the data is stored in the dummy
	if !dummy.channelID.Cmp(chID) {
		t.Errorf("The channel IDs do not match %s vs %s",
			dummy.channelID, chID)
	}

	if !dummy.messageID.Equals(msgID) {
		t.Errorf("The message IDs do not match %s vs %s",
			dummy.messageID, msgID)
	}

	if dummy.messageType != mt {
		t.Errorf("The message types do not match %s vs %s",
			dummy.messageType, mt)
	}

	if dummy.nickname != AdminUsername {
		t.Errorf("The usernames do not match %s vs %s",
			dummy.nickname, AdminUsername)
	}

	if !bytes.Equal(dummy.content, cm.Payload) {
		t.Errorf("The payloads do not match %s vs %s",
			dummy.content, cm.Payload)
	}

	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("The timestamps do not match %s vs %s",
			dummy.timestamp, r.Timestamps[states.QUEUED])
	}

	if dummy.lease != time.Duration(cm.Lease) {
		t.Errorf("The messge lease durations do not match %s vs %s",
			dummy.lease, time.Duration(cm.Lease))
	}

	if dummy.round.ID != r.ID {
		t.Errorf("The messge round does not match %s vs %s",
			dummy.round.ID, r.ID)
	}
}

func TestEvents_triggerAdminEvents_noChannel(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	dummy := &dummyMessageTypeHandler{}

	mt := MessageType(1)
	// skip handler registration

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	u, _, cm := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	msgID := cryptoChannel.MakeMessageID(u.userMessage.Message, chID)

	// call the trigger
	_, err := e.triggerAdminEvent(chID, cm, netTime.Now(), msgID,
		receptionID.EphemeralIdentity{}, r, Delivered)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// check that the event was triggered
	if dummy.triggered {
		t.Errorf("The admin event was triggered when unregistered")
	}
}
func TestEvents_triggerActionEvent(t *testing.T) {
	e := initEvents(&MockEvent{}, versioned.NewKV(ekv.MakeMemstore()))
	dummy := &dummyMessageTypeHandler{}

	// Register the handler
	mt := MessageType(42)
	err := e.RegisterReceiveHandler(mt, dummy.dummyMessageTypeReceiveMessage)
	if err != nil {
		t.Fatalf("Error on registration: %+v", err)
	}

	// Craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	u, _, cm := builtTestUMI(t, mt)

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	msgID := cryptoChannel.MakeMessageID(u.userMessage.Message, chID)

	// Call the trigger
	_, err = e.triggerActionEvent(
		chID, msgID, MessageType(cm.PayloadType), cm.Nickname, cm.Payload,
		netTime.Now(), time.Duration(cm.Lease), r, Delivered, true)
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Check that the event was triggered
	if !dummy.triggered {
		t.Error("The trigger event was not triggered")
	}

	// Check the data is stored in the dummy
	if !dummy.channelID.Cmp(chID) {
		t.Errorf("Incorrect channel ID.\nexpected: %s\nreceived: %s",
			chID, dummy.channelID)
	}
	if !dummy.messageID.Equals(msgID) {
		t.Errorf("Incorrect message ID.\nexpected: %s\nreceived: %s",
			msgID, dummy.messageID)
	}
	if dummy.messageType != mt {
		t.Errorf("Incorrect message type.\nexpected: %s\nreceived: %s",
			mt, dummy.messageType)
	}
	if dummy.nickname != cm.Nickname {
		t.Errorf("Incorrect username.\nexpected: %s\nreceived: %s",
			cm.Nickname, dummy.nickname)
	}
	if !bytes.Equal(dummy.content, cm.Payload) {
		t.Errorf("Incorrect payload.\nexpected: %q\nreceived: %q",
			cm.Payload, dummy.content)
	}
	if !withinMutationWindow(r.Timestamps[states.QUEUED], dummy.timestamp) {
		t.Errorf("Incorrect timestamps.\nexpected: %s\nreceived: %s",
			r.Timestamps[states.QUEUED], dummy.timestamp)
	}
	if dummy.lease != time.Duration(cm.Lease) {
		t.Errorf("Incorrect lease duration.\nexpected: %s\nreceived: %s",
			time.Duration(cm.Lease), dummy.lease)
	}
	if dummy.round.ID != r.ID {
		t.Errorf("Incorrect round ID.\nexpected: %d\nreceived: %d",
			r.ID, dummy.round.ID)
	}
}

func TestEvents_receiveTextMessage_Message(t *testing.T) {
	me := &MockEvent{}

	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: nil,
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf("GenerateIdentity error: %+v", err)
	}

	senderNickname := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	// call the handler
	e.receiveTextMessage(chID, msgID, 0, senderNickname, textMarshaled,
		pi.PubKey, pi.CodesetVersion, ts, lease, r, Delivered, false)

	// check the results on the model
	if !me.eventReceive.channelID.Cmp(chID) {
		t.Errorf("Channel ID did not propogate correctly, %s vs %s",
			me.eventReceive.channelID, chID)
	}

	if !me.eventReceive.messageID.Equals(msgID) {
		t.Errorf("Message ID did not propogate correctly, %s vs %s",
			me.eventReceive.messageID, msgID)
	}

	if !me.eventReceive.reactionTo.Equals(cryptoChannel.MessageID{}) {
		t.Errorf("Reaction ID is not blank, %s", me.eventReceive.reactionTo)
	}

	if me.eventReceive.nickname != senderNickname {
		t.Errorf("SenderID propogate correctly, %s vs %s",
			me.eventReceive.nickname, senderNickname)
	}

	if me.eventReceive.timestamp != ts {
		t.Errorf("Message timestamp did not propogate correctly, %s vs %s",
			me.eventReceive.timestamp, ts)
	}

	if me.eventReceive.lease != lease {
		t.Errorf("Message lease did not propogate correctly, %s vs %s",
			me.eventReceive.lease, lease)
	}

	if me.eventReceive.round.ID != r.ID {
		t.Errorf("Message round did not propogate correctly, %d vs %d",
			me.eventReceive.round.ID, r.ID)
	}
}

func TestEvents_receiveTextMessage_Reply(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	replyMsgId := cryptoChannel.MakeMessageID([]byte("blarg"), chID)

	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: replyMsgId[:],
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	senderUsername := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// call the handler
	e.receiveTextMessage(chID, msgID, Text, senderUsername, textMarshaled,
		pi.PubKey, pi.CodesetVersion, ts, lease, r, Delivered, false)

	// check the results on the model
	if !me.eventReceive.channelID.Cmp(chID) {
		t.Errorf("Channel ID did not propogate correctly, %s vs %s",
			me.eventReceive.channelID, chID)
	}

	if !me.eventReceive.messageID.Equals(msgID) {
		t.Errorf("Message ID did not propogate correctly, %s vs %s",
			me.eventReceive.messageID, msgID)
	}

	if !me.eventReceive.reactionTo.Equals(replyMsgId) {
		t.Errorf("Reaction ID is not equal to what was passed in, "+
			"%s vs %s", me.eventReceive.reactionTo, replyMsgId)
	}

	if me.eventReceive.nickname != senderUsername {
		t.Errorf("SenderID propogate correctly, %s vs %s",
			me.eventReceive.nickname, senderUsername)
	}

	if me.eventReceive.timestamp != ts {
		t.Errorf("Message timestamp did not propogate correctly, "+
			"%s vs %s", me.eventReceive.timestamp, ts)
	}

	if me.eventReceive.lease != lease {
		t.Errorf("Message lease did not propogate correctly, %s vs %s",
			me.eventReceive.lease, lease)
	}

	if me.eventReceive.round.ID != r.ID {
		t.Errorf("Message round did not propogate correctly, %d vs %d",
			me.eventReceive.round.ID, r.ID)
	}
}

func TestEvents_receiveTextMessage_Reply_BadReply(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	replyMsgId := []byte("blarg")

	textPayload := &CMIXChannelText{
		Version:        0,
		Text:           "They Don't Think It Be Like It Is, But It Do",
		ReplyMessageID: replyMsgId[:],
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	senderUsername := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// call the handler
	e.receiveTextMessage(chID, msgID, 0, senderUsername, textMarshaled,
		pi.PubKey, pi.CodesetVersion, ts, lease, r, Delivered, false)

	// check the results on the model
	if !me.eventReceive.channelID.Cmp(chID) {
		t.Errorf("Channel ID did not propogate correctly, %s vs %s",
			me.eventReceive.channelID, chID)
	}

	if !me.eventReceive.messageID.Equals(msgID) {
		t.Errorf("Message ID did not propogate correctly, %s vs %s",
			me.eventReceive.messageID, msgID)
	}

	if !me.eventReceive.reactionTo.Equals(cryptoChannel.MessageID{}) {
		t.Errorf("Reaction ID is not blank, %s", me.eventReceive.reactionTo)
	}

	if me.eventReceive.nickname != senderUsername {
		t.Errorf("SenderID propogate correctly, %s vs %s",
			me.eventReceive.nickname, senderUsername)
	}

	if me.eventReceive.timestamp != ts {
		t.Errorf("Message timestamp did not propogate correctly, %s vs %s",
			me.eventReceive.timestamp, ts)
	}

	if me.eventReceive.lease != lease {
		t.Errorf("Message lease did not propogate correctly, %s vs %s",
			me.eventReceive.lease, lease)
	}

	if me.eventReceive.round.ID != r.ID {
		t.Errorf("Message round did not propogate correctly, %d vs %d",
			me.eventReceive.round.ID, r.ID)
	}
}

func TestEvents_receiveReaction(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	replyMsgId := cryptoChannel.MakeMessageID([]byte("blarg"), chID)

	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "🍆",
		ReactionMessageID: replyMsgId[:],
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	senderUsername := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// call the handler
	e.receiveReaction(chID, msgID, 0, senderUsername, textMarshaled, pi.PubKey,
		pi.CodesetVersion, ts, lease, r, Delivered, false)

	// check the results on the model
	if !me.eventReceive.channelID.Cmp(chID) {
		t.Errorf("Channel ID did not propogate correctly, %s vs %s",
			me.eventReceive.channelID, chID)
	}

	if !me.eventReceive.messageID.Equals(msgID) {
		t.Errorf("Message ID did not propogate correctly, %s vs %s",
			me.eventReceive.messageID, msgID)
	}

	if !me.eventReceive.reactionTo.Equals(replyMsgId) {
		t.Errorf("Reaction ID is not equal to what was passed in, %s vs %s",
			me.eventReceive.reactionTo, replyMsgId)
	}

	if me.eventReceive.nickname != senderUsername {
		t.Errorf("SenderID propogate correctly, %s vs %s",
			me.eventReceive.nickname, senderUsername)
	}

	if me.eventReceive.timestamp != ts {
		t.Errorf("Message timestamp did not propogate correctly, "+
			"%s vs %s", me.eventReceive.timestamp, ts)
	}

	if me.eventReceive.lease != lease {
		t.Errorf("Message lease did not propogate correctly, %s vs %s",
			me.eventReceive.lease, lease)
	}

	if me.eventReceive.round.ID != r.ID {
		t.Errorf("Message round did not propogate correctly, %d vs %d",
			me.eventReceive.round.ID, r.ID)
	}
}

func TestEvents_receiveReaction_InvalidReactionMessageID(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	replyMsgId := []byte("blarg")

	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "🍆",
		ReactionMessageID: replyMsgId[:],
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	senderUsername := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// call the handler
	e.receiveReaction(chID, msgID, 0, senderUsername, textMarshaled, pi.PubKey,
		pi.CodesetVersion, ts, lease, r, Delivered, false)

	// check the results on the model
	if me.eventReceive.channelID != nil {
		t.Error("Channel ID did propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.messageID.Equals(msgID) {
		t.Errorf("Message ID propagate correctly when the reaction is bad.")
	}

	if !me.eventReceive.reactionTo.Equals(cryptoChannel.MessageID{}) {
		t.Errorf("Reaction ID propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.nickname != "" {
		t.Errorf("SenderID propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.lease != 0 {
		t.Errorf("Message lease propagate correctly when the reaction is bad.")
	}
}

func TestEvents_receiveReaction_InvalidReactionContent(t *testing.T) {
	me := &MockEvent{}
	e := initEvents(me, versioned.NewKV(ekv.MakeMemstore()))

	// craft the input for the event
	chID := &id.ID{}
	chID[0] = 1

	replyMsgId := cryptoChannel.MakeMessageID([]byte("blarg"), chID)

	textPayload := &CMIXChannelReaction{
		Version:           0,
		Reaction:          "I'm not a reaction",
		ReactionMessageID: replyMsgId[:],
	}

	textMarshaled, err := proto.Marshal(textPayload)
	if err != nil {
		t.Fatalf("Failed to marshael the message proto: %+v", err)
	}

	msgID := cryptoChannel.MakeMessageID(textMarshaled, chID)

	senderUsername := "Alice"
	ts := netTime.Now()

	lease := 69 * time.Minute

	r := rounds.Round{ID: 420, Timestamps: make(map[states.Round]time.Time)}
	r.Timestamps[states.QUEUED] = netTime.Now()

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	// Call the handler
	e.receiveReaction(chID, msgID, 0, senderUsername, textMarshaled, pi.PubKey,
		pi.CodesetVersion, ts, lease, r, Delivered, false)

	// Check the results on the model
	if me.eventReceive.channelID != nil {
		t.Error("Channel ID did propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.messageID.Equals(msgID) {
		t.Error("Message ID propagate correctly when the reaction is bad.")
	}

	if !me.eventReceive.reactionTo.Equals(cryptoChannel.MessageID{}) {
		t.Error("Reaction ID propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.nickname != "" {
		t.Error("SenderID propagate correctly when the reaction is bad.")
	}

	if me.eventReceive.lease != 0 {
		t.Error("Message lease propagate correctly when the reaction is bad.")
	}
}

func getFuncName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}
