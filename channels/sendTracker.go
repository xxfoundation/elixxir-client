////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	sendTrackerStorageKey     = "sendTrackerStorageKey"
	sendTrackerStorageVersion = 0

	sendTrackerUnsentStorageKey     = "sendTrackerUnsentStorageKey"
	sendTrackerUnsentStorageVersion = 0

	getRoundResultsTimeout = 60 * time.Second
	// Number of times it will attempt to get round status before the round is
	// assumed to have failed. Tracking per round does not persist across runs
	maxChecks = 3

	oneSecond = 1_000 * time.Millisecond
)

type tracked struct {
	MsgID     message.ID
	ChannelID *id.ID
	RoundID   id.Round
	UUID      uint64
}

type trackedList struct {
	List           []*tracked
	RoundCompleted bool
}

// sendTracker tracks outbound messages and denotes when they are delivered to
// the event model. It also captures incoming messages and in the event they
// were sent by this user diverts them as status updates on the previously sent
// messages.
type sendTracker struct {
	byRound     map[id.Round]trackedList
	byMessageID map[message.ID]*tracked
	unsent      map[uint64]*tracked

	trigger      triggerEventFunc
	adminTrigger triggerAdminEventFunc
	updateStatus UpdateFromUuidFunc

	net Client

	rngSrc *fastRNG.StreamGenerator
	kv     versioned.KV
	mux    sync.RWMutex
}

// messageReceiveFunc is a function type for sendTracker.MessageReceive so it
// can be mocked for testing where used.
type messageReceiveFunc func(
	messageID message.ID, r rounds.Round) bool

// loadSendTracker loads a sent tracker, restoring from disk. It will register a
// function with the cmix client, delayed on when the network goes healthy,
// which will attempt to discover the status of all rounds that are outstanding.
func loadSendTracker(net Client, kv versioned.KV, trigger triggerEventFunc,
	adminTrigger triggerAdminEventFunc, updateStatus UpdateFromUuidFunc,
	rngSource *fastRNG.StreamGenerator) *sendTracker {
	st := &sendTracker{
		byRound:      make(map[id.Round]trackedList),
		byMessageID:  make(map[message.ID]*tracked),
		unsent:       make(map[uint64]*tracked),
		trigger:      trigger,
		adminTrigger: adminTrigger,
		updateStatus: updateStatus,
		net:          net,
		rngSrc:       rngSource,
		kv:           kv,
	}

	if err := st.load(); err != nil && kv.Exists(err) {
		jww.FATAL.Panicf("[CH] Failed to load channels sent tracker: %+v", err)
	}

	// Denote all unsent messages as failed and clear
	for uuid, t := range st.unsent {
		status := Failed
		err := updateStatus(uuid, &t.MsgID, nil, nil, nil, nil, &status)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to update message %s (UUID %d): %+v",
				t.MsgID, uuid, err)
		}
	}
	st.unsent = make(map[uint64]*tracked)

	// Register to check all outstanding rounds when the network becomes healthy
	var callBackID uint64
	callBackID = net.AddHealthCallback(func(f bool) {
		if !f {
			return
		}

		net.RemoveHealthCallback(callBackID)
		for rid, oldTracked := range st.byRound {
			if oldTracked.RoundCompleted {
				continue
			}

			rr := &roundResults{
				round: rid,
				st:    st,
			}
			st.net.GetRoundResults(
				getRoundResultsTimeout, rr.callback, rr.round)
		}
	})

	return st
}

// store writes the list of rounds that have been.
func (st *sendTracker) store() error {
	if err := st.storeSent(); err != nil {
		return err
	}

	return st.storeUnsent()
}

func (st *sendTracker) storeSent() error {
	// Save sent messages
	data, err := json.Marshal(&st.byRound)
	if err != nil {
		return err
	}
	return st.kv.Set(sendTrackerStorageKey, &versioned.Object{
		Version:   sendTrackerStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}

// store writes the list of rounds that have been.
func (st *sendTracker) storeUnsent() error {
	// Save unsent messages
	data, err := json.Marshal(&st.unsent)
	if err != nil {
		return err
	}

	return st.kv.Set(sendTrackerUnsentStorageKey, &versioned.Object{
		Version:   sendTrackerUnsentStorageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	})
}

// load will get the stored rounds to be checked from disk and builds internal
// datastructures.
func (st *sendTracker) load() error {
	obj, err := st.kv.Get(sendTrackerStorageKey, sendTrackerStorageVersion)
	if err != nil {
		return err
	}

	err = json.Unmarshal(obj.Data, &st.byRound)
	if err != nil {
		return err
	}

	for rid := range st.byRound {
		roundList := st.byRound[rid].List
		for j := range roundList {
			st.byMessageID[roundList[j].MsgID] = roundList[j]
		}
	}

	obj, err = st.kv.Get(
		sendTrackerUnsentStorageKey, sendTrackerUnsentStorageVersion)
	if err != nil {
		return err
	}

	err = json.Unmarshal(obj.Data, &st.unsent)
	if err != nil {
		return err
	}

	return nil
}

// denotePendingSend is called before the pending send. It tracks the send
// internally and notifies the UI of the send.
func (st *sendTracker) denotePendingSend(channelID *id.ID,
	umi *userMessageInternal) (uint64, error) {
	// For the message timestamp, use 1 second from now to approximate the lag
	// due to round submission
	ts := netTime.Now().Add(oneSecond)

	// Create a random message ID so that there won't be collisions in a
	// database that requires a unique message ID
	stream := st.rngSrc.GetStream()
	umi.messageID = message.ID{}
	n, err := stream.Read(umi.messageID[:])
	if err != nil {
		jww.FATAL.Panicf(
			"[CH] Failed to get generate random message ID: %+v", err)
	} else if n != len(umi.messageID[:]) {
		jww.FATAL.Panicf(
			"[CH] Generated %d bytes for message ID; %d bytes required.",
			n, len(umi.messageID[:]))
	}
	stream.Close()

	// Submit the message to the UI
	uuid, err := st.trigger(channelID, umi, nil, ts,
		receptionID.EphemeralIdentity{}, rounds.Round{}, Unsent)
	if err != nil {
		return 0, err
	}

	// Track the message on disk
	st.handleDenoteSend(uuid, channelID, umi.messageID, rounds.Round{})
	return uuid, nil
}

// denotePendingAdminSend is called before the pending admin send. It tracks the
// send internally and notifies the UI of the send.
func (st *sendTracker) denotePendingAdminSend(channelID *id.ID,
	cm *ChannelMessage, encryptedPayload []byte) (uint64, error) {
	// For a timestamp for the message, use 1 second from now to approximate the
	// lag due to round submission
	ts := netTime.Now().Add(oneSecond)

	// Create a random message ID to avoid collisions in a database that
	// requires a unique message ID
	stream := st.rngSrc.GetStream()
	var randMessageID message.ID
	if n, err := stream.Read(randMessageID[:]); err != nil {
		jww.FATAL.Panicf("[CH] Failed to generate a random message ID on "+
			"channel %s: %+v", channelID, err)
	} else if n != message.IDLen {
		jww.FATAL.Panicf("[CH] Failed to generate a random message ID on "+
			"channel %s: generated %d bytes when %d bytes are required",
			channelID, n, message.IDLen)
	}
	stream.Close()

	// Submit the message to the UI
	uuid, err := st.adminTrigger(channelID, cm, 42, encryptedPayload, ts,
		randMessageID, receptionID.EphemeralIdentity{}, rounds.Round{}, Unsent)
	if err != nil {
		return 0, err
	}

	// Track the message on disk
	st.handleDenoteSend(uuid, channelID, randMessageID, rounds.Round{})
	return uuid, nil
}

// handleDenoteSend does the nitty-gritty of editing internal structures.
func (st *sendTracker) handleDenoteSend(uuid uint64, channelID *id.ID,
	messageID message.ID, round rounds.Round) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Skip if already added
	_, existsMessage := st.unsent[uuid]
	if existsMessage {
		return
	}

	st.unsent[uuid] = &tracked{messageID, channelID, round.ID, uuid}

	err := st.storeUnsent()
	if err != nil {
		jww.FATAL.Panicf("[CH] Failed to store unsent for message %s "+
			"(UUID %d) in channel %s on round %d: %+v",
			messageID, uuid, channelID, round.ID, err)
	}
}

// send tracks a generic send message.
func (st *sendTracker) send(
	uuid uint64, msgID message.ID, round rounds.Round) error {
	// Update the on disk message status
	t, err := st.handleSend(uuid, msgID, round)
	if err != nil {
		return err
	}

	// Modify the timestamp to reduce the chance message order will be ambiguous
	ts := message.MutateTimestamp(round.Timestamps[states.QUEUED], msgID)

	// Update the message in the UI
	status := Sent
	go func() {
		err = st.updateStatus(t.UUID, &msgID, &ts, &round, nil, nil, &status)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to update message %s (UUID %d): %+v",
				t.MsgID, uuid, err)
		}
	}()
	return nil
}

// send tracks a generic send message.
func (st *sendTracker) failedSend(uuid uint64) error {
	// Update the on disk message status
	t, err := st.handleSendFailed(uuid)
	if err != nil {
		return err
	}

	// Update the message in the UI
	status := Failed
	go func() {
		err = st.updateStatus(t.UUID, nil, nil, nil, nil, nil, &status)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to update message UUID %d: %+v",
				uuid, err)
		}
	}()
	return nil
}

// handleSend does the nitty-gritty of editing internal structures.
func (st *sendTracker) handleSend(uuid uint64,
	messageID message.ID, round rounds.Round) (*tracked, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Check if it is in unsent
	t, exists := st.unsent[uuid]
	if !exists {
		return nil, errors.New("cannot handle send on an unprepared message")
	}

	_, existsMessage := st.byMessageID[messageID]
	if existsMessage {
		return nil,
			errors.New("cannot handle send on a message which was already sent")
	}

	t.MsgID = messageID
	t.RoundID = round.ID

	// Add the roundID
	roundsList, existsRound := st.byRound[round.ID]
	roundsList.List = append(roundsList.List, t)
	st.byRound[round.ID] = roundsList

	// Add the round
	st.byMessageID[messageID] = t

	if !existsRound {
		rr := &roundResults{
			round: round.ID,
			st:    st,
		}
		st.net.GetRoundResults(getRoundResultsTimeout, rr.callback, rr.round)
	}

	delete(st.unsent, uuid)

	// Store the changed list to disk
	err := st.store()
	if err != nil {
		jww.FATAL.Panicf("[CH] Failed to store changes for message %s "+
			"(UUID %d) on round %d: %+v", messageID, uuid, round.ID, err)
	}

	return t, nil
}

// handleSendFailed does the nitty-gritty of editing internal structures.
func (st *sendTracker) handleSendFailed(uuid uint64) (*tracked, error) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Check if it is in unsent
	t, exists := st.unsent[uuid]
	if !exists {
		return nil, errors.New("cannot handle send on an unprepared message")
	}

	delete(st.unsent, uuid)

	// Store the changed list to disk
	err := st.storeUnsent()
	if err != nil {
		jww.FATAL.Panicf(
			"[CH] Failed to store changes for UUID %d: %+v", uuid, err)
	}

	return t, nil
}

// MessageReceive is used when a message is received to check if the message was
// sent by this user. If it was, the correct signal is sent to the event model
// and the function returns true, notifying the caller to not process the
// message.
func (st *sendTracker) MessageReceive(
	messageID message.ID, round rounds.Round) bool {
	st.mux.RLock()

	// Skip if already added
	_, existsMessage := st.byMessageID[messageID]
	st.mux.RUnlock()
	if !existsMessage {
		return false
	}

	st.mux.Lock()
	defer st.mux.Unlock()
	msgData, existsMessage := st.byMessageID[messageID]
	if !existsMessage {
		return false
	}

	delete(st.byMessageID, messageID)

	roundList := st.byRound[msgData.RoundID]
	if len(roundList.List) == 1 {
		delete(st.byRound, msgData.RoundID)
	} else {
		newRoundList := make([]*tracked, 0, len(roundList.List)-1)
		for i := range roundList.List {
			if !roundList.List[i].MsgID.Equals(messageID) {
				newRoundList = append(newRoundList, roundList.List[i])
			}
		}
		st.byRound[msgData.RoundID] = trackedList{
			List:           newRoundList,
			RoundCompleted: roundList.RoundCompleted,
		}
	}

	ts := message.MutateTimestamp(round.Timestamps[states.QUEUED], messageID)
	status := Delivered

	go func() {
		err := st.updateStatus(
			msgData.UUID, &messageID, &ts, &round, nil, nil, &status)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to update message %s (UUID %d): %+v",
				messageID, msgData.UUID, err)
		}
	}()

	if err := st.storeSent(); err != nil {
		jww.FATAL.Panicf("[CH] Failed to store the updated sent list: %+v", err)
	}

	return true
}

// roundResults represents a round which results are waiting on from the cMix
// layer.
type roundResults struct {
	round     id.Round
	st        *sendTracker
	numChecks uint
}

// callback is called when results are known about a round. it will re-trigger
// the wait if it fails up to 'maxChecks' times.
func (rr *roundResults) callback(
	allRoundsSucceeded, timedOut bool, results map[id.Round]cmix.RoundResult) {
	rr.st.mux.Lock()

	// If the message was already handled, then do nothing
	registered, existsRound := rr.st.byRound[rr.round]
	if !existsRound {
		rr.st.mux.Unlock()
		return
	}

	status := Delivered
	if !allRoundsSucceeded {
		status = Failed
	}

	if timedOut {
		if rr.numChecks >= maxChecks {
			jww.WARN.Printf("[CH] Channel messages sent on %d assumed to "+
				"have failed after %d attempts to get round status", rr.round,
				maxChecks)
			status = Failed
		} else {
			rr.numChecks++

			rr.st.mux.Unlock()

			// Retry if timed out
			go rr.st.net.GetRoundResults(
				getRoundResultsTimeout, rr.callback, []id.Round{rr.round}...)
			return
		}
	}

	registered.RoundCompleted = true
	rr.st.byRound[rr.round] = registered
	if err := rr.st.store(); err != nil {
		jww.FATAL.Panicf("[CH] Failed to store update after finalizing "+
			"delivery of sent messages: %+v", err)
	}

	rr.st.mux.Unlock()
	if status == Failed {
		for i := range registered.List {
			round := results[rr.round].Round
			status = Failed
			go func(i int, round rounds.Round, status SentStatus) {
				err := rr.st.updateStatus(registered.List[i].UUID,
					&registered.List[i].MsgID, nil, &round, nil, nil, &status)
				if err != nil {
					jww.ERROR.Printf(
						"[CH] Failed to update message %s (UUID %d): %+v",
						registered.List[i].MsgID, registered.List[i].UUID, err)
				}
			}(i, round, status)
		}
	}
}
