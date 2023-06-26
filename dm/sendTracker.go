////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

// sendTracker is a copy of the one from Channels, but with required differences
// as we couldn't use the one from channels off the shelf

package dm

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	sendTrackerStorageKey     = "dmSendTrackerStorageKey"
	sendTrackerStorageVersion = 0

	sendTrackerUnsentStorageKey     = "dmSendTrackerUnsentStorageKey"
	sendTrackerUnsentStorageVersion = 0

	getRoundResultsTimeout = 60 * time.Second
	// Number of times it will attempt to get round status before
	// the round is assumed to have failed. Tracking per round
	// does not persist across runs
	maxChecks = 3

	oneSecond = 1000 * time.Millisecond
)

type triggerEventFunc func(msgID message.ID, messageType MessageType,
	nick string, plaintext []byte, dmToken uint32,
	partnerPubKey, senderPubKey ed25519.PublicKey, ts time.Time,
	_ receptionID.EphemeralIdentity, round rounds.Round,
	status Status) (uint64, error)

type updateStatusFunc func(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status Status)

type tracked struct {
	MsgID   message.ID `json:"msgID"`
	RoundID id.Round   `json:"roundID"`
	UUID    uint64     `json:"uuid"`

	// For logging/debugging purposes
	partnerKey ed25519.PublicKey
	senderKey  ed25519.PublicKey
}

type trackedList struct {
	List           []*tracked `json:"list"`
	RoundCompleted bool       `json:"roundCompleted"`
}

// sendTracker tracks outbound messages and denotes when they are delivered to
// the event model. It also captures incoming messages and, in the event they
// were sent by this user, diverts them as status updates on the previously sent
// messages.
type sendTracker struct {
	byRound map[id.Round]trackedList

	byMessageID map[message.ID]*tracked

	unsent map[uint64]*tracked

	mux sync.RWMutex

	trigger      triggerEventFunc
	updateStatus updateStatusFunc

	net cMixClient

	kv versioned.KV

	rngSrc *fastRNG.StreamGenerator
}

// NewSendTracker returns an uninitialized SendTracker object. The DM
// Client will call Init to initialize it.
func NewSendTracker(kv versioned.KV) SendTracker {
	return &sendTracker{kv: kv}
}

// messageReceiveFunc is a function type for sendTracker.MessageReceive so it
// can be mocked for testing where used.
type messageReceiveFunc func(
	messageID message.ID, r rounds.Round) bool

// Init loads a sent tracker, restoring from disk. It will register a
// function with the cmix client, delayed on when the network goes healthy,
// which will attempt to discover the status of all rounds that are outstanding.
func (st *sendTracker) Init(net cMixClient, trigger triggerEventFunc,
	updateStatus updateStatusFunc,
	rngSource *fastRNG.StreamGenerator) {
	st.byRound = make(map[id.Round]trackedList)
	st.byMessageID = make(map[message.ID]*tracked)
	st.unsent = make(map[uint64]*tracked)
	st.trigger = trigger
	st.updateStatus = updateStatus
	st.net = net
	st.rngSrc = rngSource

	if err := st.load(); err != nil && st.kv.Exists(err) {
		jww.FATAL.Panicf("Failed to load channels sent tracker: %+v",
			err)
	}

	// Denote all unsent messages as failed and clear
	for uuid, t := range st.unsent {
		updateStatus(uuid, t.MsgID, time.Time{}, rounds.Round{}, Failed)
	}
	st.unsent = make(map[uint64]*tracked)

	// Register to check all outstanding rounds when the network
	// becomes healthy
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
}

// DenotePendingSend is called before the pending send. It tracks the send
// internally and notifies the UI of the send.
func (st *sendTracker) DenotePendingSend(partnerPubKey, senderPubKey ed25519.PublicKey,
	partnerToken uint32, messageType MessageType,
	dm *DirectMessage) (uint64, error) {
	// For the message timestamp, use 1 second from now to
	// approximate the lag due to round submission
	ts := netTime.Now().Add(oneSecond)

	// Create a random message ID so that there won't be collisions in a
	// database that requires a unique message ID
	stream := st.rngSrc.GetStream()
	messageID := message.ID{}
	n, err := stream.Read(messageID[:])
	if err != nil {
		jww.FATAL.Panicf(
			"Failed to get generate random message ID: %+v", err)
	} else if n != len(messageID[:]) {
		jww.FATAL.Panicf(
			"Generated %d bytes for message ID; %d bytes required.",
			n, len(messageID[:]))
	}
	stream.Close()

	// Submit the message to the UI
	uuid, err := st.trigger(messageID, messageType, dm.Nickname, dm.Payload,
		partnerToken, partnerPubKey, senderPubKey, ts,
		receptionID.EphemeralIdentity{},
		rounds.Round{}, Unsent)
	if err != nil {
		return 0, err
	}

	// Track the message on disk
	st.handleDenoteSend(uuid, partnerPubKey, senderPubKey, messageID,
		rounds.Round{})

	return uuid, nil
}

// handleDenoteSend does the nitty-gritty of editing internal structures.
func (st *sendTracker) handleDenoteSend(uuid uint64,
	partnerKey, senderKey ed25519.PublicKey,
	messageID message.ID, round rounds.Round) {
	st.mux.Lock()
	defer st.mux.Unlock()

	// Skip if already added
	_, existsMessage := st.unsent[uuid]
	if existsMessage {
		return
	}

	st.unsent[uuid] = &tracked{messageID, round.ID, uuid, partnerKey, senderKey}

	err := st.storeUnsent()
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
}

// Sent tracks a generic send message.
func (st *sendTracker) Sent(
	uuid uint64, msgID message.ID, round rounds.Round) error {
	// Update the on disk message status
	t, err := st.handleSend(uuid, msgID, round)
	if err != nil {
		return err
	}

	// Modify the timestamp to reduce the chance message order
	// will be ambiguous
	ts := message.MutateTimestamp(round.Timestamps[states.QUEUED], msgID)

	// Update the message in the UI
	go st.updateStatus(t.UUID, msgID, ts, round, Sent)
	return nil
}

// FailedSend marks the message send as failed.
func (st *sendTracker) FailedSend(uuid uint64) error {
	// Update the on disk message status
	t, err := st.handleSendFailed(uuid)
	if err != nil {
		return err
	}

	// Update the message in the UI
	go st.updateStatus(
		t.UUID, message.ID{}, time.Time{}, rounds.Round{}, Failed)
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
		return nil, errors.New(
			"cannot handle send on an unprepared message")
	}

	_, existsMessage := st.byMessageID[messageID]
	if existsMessage {
		return nil,
			errors.New("cannot handle send on a message which was" +
				" already sent")
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
		st.net.GetRoundResults(getRoundResultsTimeout, rr.callback,
			rr.round)
	}

	delete(st.unsent, uuid)

	// Store the changed list to disk
	err := st.store()
	if err != nil {
		jww.FATAL.Panicf(err.Error())
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
		return nil, errors.New(
			"cannot handle send on an unprepared message")
	}

	delete(st.unsent, uuid)

	// Store the changed list to disk
	err := st.storeUnsent()
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}

	return t, nil
}

// CheckIfSent is used when a message is received to check if the message was
// sent by this user.
func (st *sendTracker) CheckIfSent(messageID message.ID, _ rounds.Round) bool {
	st.mux.RLock()
	defer st.mux.RUnlock()
	_, existsMessage := st.byMessageID[messageID]
	return existsMessage
}

// Delivered calls the event model update function to tell it that this
// message was delivered. (after this is called successfully, it is safe to
// stop tracking this message).
// returns true if the update sent status func was called.
func (st *sendTracker) Delivered(messageID message.ID,
	round rounds.Round) bool {
	st.mux.RLock()
	defer st.mux.RUnlock()
	msgData, existsMessage := st.byMessageID[messageID]
	if !existsMessage {
		return false
	}

	ts := message.MutateTimestamp(round.Timestamps[states.QUEUED],
		messageID)
	st.updateStatus(msgData.UUID, messageID, ts,
		round, Received)
	return true
}

// StopTracking deletes this message id/round combination from the
// send tracking.  returns true if it was removed, false otherwise.
func (st *sendTracker) StopTracking(messageID message.ID, _ rounds.Round) bool {
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
				newRoundList = append(newRoundList,
					roundList.List[i])
			}
		}
		st.byRound[msgData.RoundID] = trackedList{
			List:           newRoundList,
			RoundCompleted: roundList.RoundCompleted,
		}
	}

	if err := st.storeSent(); err != nil {
		jww.FATAL.Panicf("failed to store the updated sent list: %+v",
			err)
	}

	return true
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

	status := Sent
	if !allRoundsSucceeded {
		status = Failed
	}

	if timedOut {
		if rr.numChecks >= maxChecks {
			jww.WARN.Printf("Channel messages sent on %d"+
				" assumed to have failed after "+
				"%d attempts to get round status", rr.round,
				maxChecks)
			status = Failed
		} else {
			rr.numChecks++

			rr.st.mux.Unlock()

			// Retry if timed out
			go rr.st.net.GetRoundResults(
				getRoundResultsTimeout, rr.callback,
				[]id.Round{rr.round}...)
			return
		}

	}

	registered.RoundCompleted = true
	rr.st.byRound[rr.round] = registered
	if err := rr.st.store(); err != nil {
		jww.FATAL.Panicf("failed to store update after "+
			"finalizing delivery of sent messages: %+v", err)
	}

	rr.st.mux.Unlock()
	if status == Failed {
		for i := range registered.List {
			round := results[rr.round].Round
			go rr.st.updateStatus(registered.List[i].UUID,
				registered.List[i].MsgID, time.Time{},
				round, Failed)
		}
	}
}
