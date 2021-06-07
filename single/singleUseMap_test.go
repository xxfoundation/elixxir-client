///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Happy path: trigger the exit channel.
func Test_pending_addState_ExitChan(t *testing.T) {
	p := newPending()
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	maxMsgs := uint8(6)
	timeout := 500 * time.Millisecond
	callback, callbackChan := createReplyComm()

	quitChan, quit, err := p.addState(rid, dhKey, maxMsgs, callback, timeout)
	if err != nil {
		t.Errorf("addState() returned an error: %+v", err)
	}

	hasQuit := atomic.CompareAndSwapInt32(quit, 0, 1)
	if !hasQuit {
		t.Error("Quit atomic called.")
	}

	expectedState := newState(dhKey, maxMsgs, callback)

	quitChan <- struct{}{}

	timer := time.NewTimer(timeout + 1*time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the quit channel was used."+
			"\npayload: %+v\nerror:   %+v", results.payload, results.err)
	case <-timer.C:
	}

	state, exists := p.singleUse[*rid]
	if !exists {
		t.Error("State not found in map.")
	}
	if !equalState(*expectedState, *state, t) {
		t.Errorf("State in map is incorrect.\nexpected: %+v\nreceived: %+v",
			*expectedState, *state)
	}

	hasQuit = atomic.CompareAndSwapInt32(quit, 0, 1)
	if hasQuit {
		t.Error("Quit atomic not called.")
	}
}

// Happy path: state is removed before deletion can occur.
func Test_pending_addState_StateRemoved(t *testing.T) {
	p := newPending()
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	maxMsgs := uint8(6)
	timeout := 5 * time.Millisecond
	callback, callbackChan := createReplyComm()

	_, _, err := p.addState(rid, dhKey, maxMsgs, callback, timeout)
	if err != nil {
		t.Errorf("addState() returned an error: %+v", err)
	}

	p.Lock()
	delete(p.singleUse, *rid)
	p.Unlock()

	timer := time.NewTimer(timeout + 1*time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback should not have been called.\npayload: %+v\nerror:   %+v",
			results.payload, results.err)
	case <-timer.C:
	}
}

// Error path: timeout occurs and deletes the entry from the map.
func Test_pending_addState_TimeoutError(t *testing.T) {
	p := newPending()
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	maxMsgs := uint8(6)
	timeout := 5 * time.Millisecond
	callback, callbackChan := createReplyComm()

	_, _, err := p.addState(rid, dhKey, maxMsgs, callback, timeout)
	if err != nil {
		t.Errorf("addState() returned an error: %+v", err)
	}

	expectedState := newState(dhKey, maxMsgs, callback)
	p.Lock()
	state, exists := p.singleUse[*rid]
	p.Unlock()
	if !exists {
		t.Error("State not found in map.")
	}
	if !equalState(*expectedState, *state, t) {
		t.Errorf("State in map is incorrect.\nexpected: %+v\nreceived: %+v",
			*expectedState, *state)
	}

	timer := time.NewTimer(timeout * 4)

	select {
	case results := <-callbackChan:
		state, exists = p.singleUse[*rid]
		if exists {
			t.Errorf("State found in map when it should have been deleted."+
				"\nstate: %+v", state)
		}
		if results.payload != nil {
			t.Errorf("Payload not nil on timeout.\npayload: %+v", results.payload)
		}
		if results.err == nil || !strings.Contains(results.err.Error(), "timed out") {
			t.Errorf("Callback did not return a time out error on return: %+v", results.err)
		}
	case <-timer.C:
		t.Error("Failed to time out.")
	}
}

// Error path: state already exists.
func Test_pending_addState_StateExistsError(t *testing.T) {
	p := newPending()
	rid := id.NewIdFromString("test RID", id.User, t)
	dhKey := getGroup().NewInt(5)
	maxMsgs := uint8(6)
	timeout := 5 * time.Millisecond
	callback, _ := createReplyComm()

	quitChan, _, err := p.addState(rid, dhKey, maxMsgs, callback, timeout)
	if err != nil {
		t.Errorf("addState() returned an error: %+v", err)
	}
	quitChan <- struct{}{}

	quitChan, _, err = p.addState(rid, dhKey, maxMsgs, callback, timeout)
	if !check(err, "a state already exists in the map") {
		t.Errorf("addState() did not return an error when the state already "+
			"exists: %+v", err)
	}
}

type replyCommData struct {
	payload []byte
	err     error
}

func createReplyComm() (func(payload []byte, err error), chan replyCommData) {
	callbackChan := make(chan replyCommData)
	callback := func(payload []byte, err error) {
		callbackChan <- replyCommData{
			payload: payload,
			err:     err,
		}
	}
	return callback, callbackChan
}

// equalState determines if the two states have equal values.
func equalState(a, b state, t *testing.T) bool {
	if a.dhKey.Cmp(b.dhKey) != 0 {
		t.Errorf("DH Keys differ.\nexpected: %s\nreceived: %s",
			a.dhKey.Text(10), b.dhKey.Text(10))
		return false
	}
	if !reflect.DeepEqual(a.fpMap.fps, b.fpMap.fps) {
		t.Errorf("Fingerprint maps differ.\nexpected: %+v\nreceived: %+v",
			a.fpMap.fps, b.fpMap.fps)
		return false
	}
	if !reflect.DeepEqual(b.c, b.c) {
		t.Errorf("collators differ.\nexpected: %+v\nreceived: %+v",
			a.c, b.c)
		return false
	}
	if reflect.ValueOf(a.callback).Pointer() != reflect.ValueOf(b.callback).Pointer() {
		t.Errorf("callbackFuncs differ.\nexpected: %p\nreceived: %p",
			a.callback, b.callback)
		return false
	}
	return true
}

// check returns true if the error is not nil and contains the substring.
func check(err error, subStr string) bool {
	return err != nil && strings.Contains(err.Error(), subStr)
}
