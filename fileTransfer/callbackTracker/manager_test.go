////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package callbackTracker

import (
	"errors"
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"io"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

// Tests that NewManager returns the expected Manager.
func TestNewManager(t *testing.T) {
	expected := &Manager{
		callbacks: make(map[ftCrypto.TransferID][]*callbackTracker),
		stops:     make(map[ftCrypto.TransferID]*stoppable.Multi),
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
	tid := &ftCrypto.TransferID{5}
	m.AddCallback(tid, cb, 0)

	// Check that the callback was called
	select {
	case <-cbChan:
	case <-time.After(25 * time.Millisecond):
		t.Error("Timed out waiting for callback to be called.")
	}

	// Check that the callback was added
	if _, exists := m.callbacks[*tid]; !exists {
		t.Errorf("No callback list found for transfer ID %s.", tid)
	} else if len(m.callbacks[*tid]) != 1 {
		t.Errorf("Incorrect number of callbacks.\nexpected: %d\nreceived: %d",
			1, len(m.callbacks[*tid]))
	}

	// Check that the stoppable was added
	if _, exists := m.stops[*tid]; !exists {
		t.Errorf("No stoppable list found for transfer ID %s.", tid)
	}
}

// Tests that Manager.Call calls al the callbacks associated with the transfer
// ID.
func TestManager_Call(t *testing.T) {
	m := NewManager()
	tid := &ftCrypto.TransferID{5}
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
		m.AddCallback(tid, cbs[i], 0)

		// Receive channel from first call
		select {
		case <-cbChans[i]:
		case <-time.After(25 * time.Millisecond):
			t.Errorf("Callback #%d never called.", i)
		}
	}

	// Call callbacks
	m.Call(tid, errors.New("test"))

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
	tid := &ftCrypto.TransferID{5}
	m.AddCallback(tid, cb, 0)

	m.Delete(tid)

	// Check that the callback was deleted
	if _, exists := m.callbacks[*tid]; exists {
		t.Errorf("Callback list found for transfer ID %s.", tid)
	}

	// Check that the stoppable was deleted
	if _, exists := m.stops[*tid]; exists {
		t.Errorf("Stoppable list found for transfer ID %s.", tid)
	}
}

// Consistency test of makeStoppableName.
func Test_makeStoppableName_Consistency(t *testing.T) {
	rng := NewPrng(42)
	expectedValues := []string{
		"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=/0",
		"39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=/1",
		"CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=/2",
		"uoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=/3",
		"GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=/4",
		"rnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=/5",
		"ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=/6",
		"SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=/7",
		"NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=/8",
		"kM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog=/9",
	}

	for i, expected := range expectedValues {
		tid, err := ftCrypto.NewTransferID(rng)
		if err != nil {
			t.Errorf("Failed to generated transfer ID #%d: %+v", i, err)
		}

		name := makeStoppableName(&tid, i)
		if expected != name {
			t.Errorf("Stoppable name does not match expected."+
				"\nexpected: %q\nreceived: %q", expected, name)
		}
	}
}

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }
