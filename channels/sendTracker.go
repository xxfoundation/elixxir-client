////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const (
	sendTrackerStorageKey     = "sendTrackerStorageKey"
	sendTrackerStorageVersion = 0
	getRoundResultsTimeout    = 60 * time.Second
	// number of times it will attempt to get round status before the round
	// is assumed to have failed. Tracking per round does not persist across
	// runs
	maxChecks = 3
)

type tracked struct {
	MsgID     cryptoChannel.MessageID
	ChannelID *id.ID
	RoundID   id.Round
}

// the sendTracker tracks outbound messages and denotes when they are delivered
// to the event model. It also captures incoming messages and in the event they
// were sent by this user diverts them as status updates on the previously sent
// messages
type sendTracker struct {
	byRound map[id.Round][]*tracked

	byMessageID map[cryptoChannel.MessageID]*tracked

	mux sync.RWMutex

	trigger      triggerEventFunc
	adminTrigger triggerAdminEventFunc
	updateStatus updateStatusFunc

	net Client

	kv *versioned.KV
}

// messageReceiveFunc is a function type for sendTracker.MessageReceive so it
// can be mocked for testing where used
type messageReceiveFunc func(messageID cryptoChannel.MessageID) bool

// loadSendTracker loads a sent tracker, restoring from disk. It will register a
// function with the cmix client, delayed on when the network goes healthy,
// which will attempt to discover the status of all rounds that are outstanding.
func loadSendTracker(net Client, kv *versioned.KV, trigger triggerEventFunc,
	adminTrigger triggerAdminEventFunc,
	updateStatus updateStatusFunc) *sendTracker {
	st := &sendTracker{
		byRound:      make(map[id.Round][]*tracked),
		byMessageID:  make(map[cryptoChannel.MessageID]*tracked),
		trigger:      trigger,
		adminTrigger: adminTrigger,
		updateStatus: updateStatus,
		net:          net,
		kv:           kv,
	}

	/*if err := st.load(); !kv.Exists(err){
		jww.FATAL.Panicf("failed to load sent tracker: %+v", err)
	}*/
	st.load()

	//register to check all outstanding rounds when the network becomes healthy
	var callBackID uint64
	callBackID = net.AddHealthCallback(func(f bool) {
		if !f {
			return
		}
		net.RemoveHealthCallback(callBackID)
		for rid := range st.byRound {
			rr := &roundResults{
				round: rid,
				st:    st,
			}
			st.net.GetRoundResults(getRoundResultsTimeout, rr.callback, rr.round)
		}
	})

	return st
}

// store writes the list of rounds that have been
func (st *sendTracker) store() error {
	data, err := json.Marshal(&st.byRound)
	if err != nil {
		return err
	}
	return st.kv.Set(sendTrackerStorageKey, &versioned.Object{
		Version:   sendTrackerStorageVersion,
		Timestamp: time.Now(),
		Data:      data,
	})
}

// load will get the stored rounds to be checked from disk and builds
// internal datastructures
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
		roundList := st.byRound[rid]
		for j := range roundList {
			st.byMessageID[roundList[j].MsgID] = roundList[j]
		}
	}
	return nil
}

// send tracks a generic send message
func (st *sendTracker) send(channelID *id.ID,
	umi *userMessageInternal, round rounds.Round) {
	st.handleSend(channelID, umi.GetMessageID(), round)
	go st.trigger(channelID, umi,
		receptionID.EphemeralIdentity{}, round, Sent)
}

// sendAdmin tracks a generic sendAdmin message
func (st *sendTracker) sendAdmin(channelID *id.ID,
	cm *ChannelMessage, msgID cryptoChannel.MessageID, round rounds.Round) {
	st.handleSend(channelID, msgID, round)
	go st.adminTrigger(channelID, cm, msgID,
		receptionID.EphemeralIdentity{}, round, Sent)
}

// handleSend does the nity gritty of editing internal structures
func (st *sendTracker) handleSend(channelID *id.ID,
	messageID cryptoChannel.MessageID, round rounds.Round) {
	st.mux.Lock()
	defer st.mux.Unlock()

	//skip if already added
	_, existsMessage := st.byMessageID[messageID]
	if existsMessage {
		return
	}

	t := &tracked{messageID, channelID, round.ID}

	//add the roundID
	roundsList, existsRound := st.byRound[round.ID]
	st.byRound[round.ID] = append(roundsList, t)

	//add the round
	st.byMessageID[messageID] = t

	if !existsRound {
		rr := &roundResults{
			round: round.ID,
			st:    st,
		}
		st.net.GetRoundResults(getRoundResultsTimeout, rr.callback, rr.round)
	}

	//store the changed list to disk
	err := st.store()
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
}

// MessageReceive is used when a message is received to check if the message
// was sent by this user. If it was, the correct signal is sent to the event
// model and the function returns true, notifying the caller to not process
// the message
func (st *sendTracker) MessageReceive(messageID cryptoChannel.MessageID) bool {
	st.mux.RLock()

	//skip if already added
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
	if len(roundList) == 1 {
		delete(st.byRound, msgData.RoundID)
	} else {
		newRoundList := make([]*tracked, 0, len(roundList)-1)
		for i := range roundList {
			if !roundList[i].MsgID.Equals(messageID) {
				newRoundList = append(newRoundList, roundList[i])
			}
		}
		st.byRound[msgData.RoundID] = newRoundList
	}

	return true
}

// roundResults represents a round which results are waiting on from the cmix layer
type roundResults struct {
	round     id.Round
	st        *sendTracker
	numChecks uint
}

// callback is called when results are known about a round. it will re-trigger
// the wait if it fails up to 'maxChecks' times.
func (rr *roundResults) callback(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {

	rr.st.mux.Lock()

	//if the message was already handled, do nothing
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
			jww.WARN.Printf("Channel messages sent on %d assumed to "+
				"have failed after %d attempts to get round status", rr.round,
				maxChecks)
			status = Failed
		} else {
			rr.numChecks++

			rr.st.mux.Unlock()

			//retry if timed out
			go rr.st.net.GetRoundResults(getRoundResultsTimeout, rr.callback, []id.Round{rr.round}...)
			return
		}

	}

	delete(rr.st.byRound, rr.round)

	for i := range registered {
		delete(rr.st.byMessageID, registered[i].MsgID)
	}

	rr.st.mux.Unlock()

	for i := range registered {
		go rr.st.updateStatus(registered[i].MsgID, status)
	}
}
