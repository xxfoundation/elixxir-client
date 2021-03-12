///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"testing"
)

// Happy path.
func Test_callbackMap_registerCallback(t *testing.T) {
	m := newTestManager(0, false, t)
	callbackChan := make(chan int)
	testCallbacks := []struct {
		tag string
		cb  ReceiveComm
	}{
		{"tag1", func([]byte, Contact) { callbackChan <- 0 }},
		{"tag2", func([]byte, Contact) { callbackChan <- 1 }},
		{"tag3", func([]byte, Contact) { callbackChan <- 2 }},
	}

	for _, val := range testCallbacks {
		m.callbackMap.registerCallback(val.tag, val.cb)
	}

	for i, val := range testCallbacks {
		go m.callbackMap.callbacks[singleUse.NewTagFP(val.tag)](nil, Contact{})
		result := <-callbackChan
		if result != i {
			t.Errorf("getCallback() did not return the expected callback."+
				"\nexpected: %d\nreceived: %d", i, result)
		}
	}
}

// Happy path.
func Test_callbackMap_getCallback(t *testing.T) {
	m := newTestManager(0, false, t)
	callbackChan := make(chan int)
	testCallbacks := []struct {
		tagFP singleUse.TagFP
		cb    ReceiveComm
	}{
		{singleUse.UnmarshalTagFP([]byte("tag1")), func([]byte, Contact) { callbackChan <- 0 }},
		{singleUse.UnmarshalTagFP([]byte("tag2")), func([]byte, Contact) { callbackChan <- 1 }},
		{singleUse.UnmarshalTagFP([]byte("tsg3")), func([]byte, Contact) { callbackChan <- 2 }},
	}

	for _, val := range testCallbacks {
		m.callbackMap.callbacks[val.tagFP] = val.cb
	}

	cb, err := m.callbackMap.getCallback(testCallbacks[1].tagFP)
	if err != nil {
		t.Errorf("getCallback() returned an error: %+v", err)
	}

	go cb(nil, Contact{})

	result := <-callbackChan
	if result != 1 {
		t.Errorf("getCallback() did not return the expected callback."+
			"\nexpected: %d\nreceived: %d", 1, result)
	}
}

// Error path: no callback exists for the given tag fingerprint.
func Test_callbackMap_getCallback_NoCallbackError(t *testing.T) {
	m := newTestManager(0, false, t)

	_, err := m.callbackMap.getCallback(singleUse.UnmarshalTagFP([]byte("tag1")))
	if err == nil {
		t.Error("getCallback() failed to return an error for a callback that " +
			"does not exist.")
	}
}
