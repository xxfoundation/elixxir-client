////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	"errors"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"gitlab.com/elixxir/client/v4/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// Tests that NewManager returns the expected Manager.
func TestNewManager(t *testing.T) {
	expected := &Manager{
		callbacks: make(map[ftCrypto.ID][]*callbackTracker),
		stops:     make(map[ftCrypto.ID]*stoppable.Multi),
	}

	newManager := NewManager()

	if !reflect.DeepEqual(expected, newManager) {
		t.Errorf("New Manager does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, newManager)
	}
}

// Tests that Manager.AddCallback adds the callback to the list, creates and
// adds a stoppable to the list, and that the callback is called.
func TestManager_AddCallback(t *testing.T) {
	m := NewManager()

	cbChan := make(chan error, 10)
	cb := func(err error) { cbChan <- err }
	fid := ftCrypto.ID{5}
	m.AddCallback(fid, cb, 0)

	// Check that the callback was called
	select {
	case <-cbChan:
	case <-time.After(25 * time.Millisecond):
		t.Error("Timed out waiting for callback to be called.")
	}

	// Check that the callback was added
	if _, exists := m.callbacks[fid]; !exists {
		t.Errorf("No callback list found for file ID %s.", fid)
	} else if len(m.callbacks[fid]) != 1 {
		t.Errorf("Incorrect number of callbacks.\nexpected: %d\nreceived: %d",
			1, len(m.callbacks[fid]))
	}

	// Check that the stoppable was added
	if _, exists := m.stops[fid]; !exists {
		t.Errorf("No stoppable list found for file ID %s.", fid)
	}
}

// Tests that Manager.Call calls al the callbacks associated with the transfer
// ID.
func TestManager_Call(t *testing.T) {
	m := NewManager()
	fid := ftCrypto.ID{5}
	n := 10
	cbChans := make([]chan error, n)
	cbs := make([]func(err error), n)
	for i := range cbChans {
		cbChan := make(chan error, 10)
		cbs[i] = func(err error) { cbChan <- err }
		cbChans[i] = cbChan
	}

	// Add callbacks
	for i := range cbs {
		m.AddCallback(fid, cbs[i], 0)

		// Receive channel from first call
		select {
		case <-cbChans[i]:
		case <-time.After(25 * time.Millisecond):
			t.Errorf("Callback #%d never called.", i)
		}
	}

	// Call callbacks
	m.Call(fid, errors.New("test"))

	// Check to make sure callbacks were called
	var wg sync.WaitGroup
	for i := range cbs {
		wg.Add(1)
		go func(i int) {
			select {
			case r := <-cbChans[i]:
				if r == nil {
					t.Errorf("Callback #%d did not receive an error.", i)
				}
			case <-time.After(25 * time.Millisecond):
				t.Errorf("Callback #%d never called.", i)
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}

// Tests that Manager.Delete removes all callbacks and stoppables from the list.
func TestManager_Delete(t *testing.T) {
	m := NewManager()

	cbChan := make(chan error, 10)
	cb := func(err error) { cbChan <- err }
	fid := ftCrypto.ID{5}
	m.AddCallback(fid, cb, 0)

	m.Delete(fid)

	// Check that the callback was deleted
	if _, exists := m.callbacks[fid]; exists {
		t.Errorf("Callback list found for file ID %s.", fid)
	}

	// Check that the stoppable was deleted
	if _, exists := m.stops[fid]; exists {
		t.Errorf("Stoppable list found for file ID %s.", fid)
	}
}

// Consistency test of makeStoppableName.
func Test_makeStoppableName_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(324))
	fileData := []byte("fileData")
	expectedValues := []string{
		"RWGYe0gv03Ydjh5Cr21KCLZRxxDzOX7bghgLbji1hG0=/0",
		"oI4SWLEAaseXW5Jz2umQlkLMcOn7KmOTvkUYKnJwWkI=/1",
		"KYPRTjrwr/bWsbp9nvF8h1VzO5LoQ8gMjie/mMueZpM=/2",
		"DlmG2f0h0h4TKcHI7tituOqgAiQ+qhqwRkB/fH2IigU=/3",
		"7vQ3s5QjD9Bwrqyy19scENj+MrA2g5i88i4GCsLUXOc=/4",
		"i+shiGVpjMUU/Fxx2bu5fLR+ypd0Mf0TmCamvNy8K5E=/5",
		"ax5kyV5oO0fwPhUVJq7jcbtth70SSdDUg2UKpzAM9nA=/6",
		"+1GvF4Dn3BB5wie8+vfMMhOYxgRmOxLCETnQb/dOoyw=/7",
		"QjbQyrLJNlP4Rp5p8Xaa66FnpwuhMmcy27z3/M6w3Ik=/8",
		"NIwj9ng0zNl6JXFtUiMiBiV8Orp4hXMd2avfgxytTJk=/9",
	}

	for i, expected := range expectedValues {
		prng.Read(fileData)
		fid := ftCrypto.NewID(fileData)

		name := makeStoppableName(fid, i)
		if expected != name {
			t.Errorf("Stoppable name does not match expected."+
				"\nexpected: %q\nreceived: %q", expected, name)
		}
	}
}
