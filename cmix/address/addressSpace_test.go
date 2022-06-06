package address

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

var initSize uint8 = 8

// Unit test of NewAddressSpace.
func TestNewAddressSpace(t *testing.T) {
	expected := &space{
		size:      initSize,
		notifyMap: make(map[string]chan uint8),
		cond:      sync.NewCond(&sync.Mutex{}),
	}

	as := NewAddressSpace(initSize)

	if !reflect.DeepEqual(expected, as) {
		t.Errorf("NewAddressSpace failed to return the expected Space."+
			"\nexpected: %+v\nreceived: %+v", expected, as)
	}
}

// Test that Space.GetAddressSpace blocks when the address space size has not
// been set and that it does not block when it has been set.
func TestSpace_GetAddressSpace(t *testing.T) {
	as := NewAddressSpace(8)
	expectedSize := uint8(42)

	// Call get and error if it does not block
	wait := make(chan uint8)
	go func() { wait <- as.GetAddressSpace() }()
	select {
	case size := <-wait:
		t.Errorf("get failed to block and returned size %d.", size)
	case <-time.NewTimer(10 * time.Millisecond).C:
	}

	// Update address size
	as.(*space).cond.L.Lock()
	as.(*space).size = expectedSize
	as.(*space).set = true
	as.(*space).cond.L.Unlock()

	// Call get and error if it does block
	wait = make(chan uint8)
	go func() { wait <- as.GetAddressSpace() }()
	select {
	case size := <-wait:
		if size != expectedSize {
			t.Errorf("get returned the wrong size.\nexpected: %d\nreceived: %d",
				expectedSize, size)
		}
	case <-time.NewTimer(150 * time.Millisecond).C:
		t.Error("get blocking when the size has been updated.")
	}
}

// Test that Space.GetAddressSpace blocks until the condition broadcasts.
func TestSpace_GetAddressSpace_WaitBroadcast(t *testing.T) {
	as := NewAddressSpace(initSize)

	wait := make(chan uint8)
	go func() { wait <- as.GetAddressSpace() }()

	go func() {
		select {
		case size := <-wait:
			if size != initSize {
				t.Errorf("get returned the wrong size.\nexpected: %d\nreceived: %d",
					initSize, size)
			}
		case <-time.NewTimer(25 * time.Millisecond).C:
			t.Error("get blocking when the Cond has broadcast.")
		}
	}()

	time.Sleep(5 * time.Millisecond)

	as.(*space).cond.Broadcast()
}

// Unit test of Space.GetAddressSpaceWithoutWait.
func TestSpace_GetAddressSpaceWithoutWait(t *testing.T) {
	as := NewAddressSpace(initSize)

	size := as.GetAddressSpaceWithoutWait()
	if size != initSize {
		t.Errorf("GetAddressSpaceWithoutWait returned the wrong size."+
			"\nexpected: %d\nreceived: %d", initSize, size)
	}
}

// Tests that Space.UpdateAddressSpace only updates the size when it is larger.
func TestSpace_UpdateAddressSpace(t *testing.T) {
	as := NewAddressSpace(initSize)
	expectedSize := uint8(42)

	// Attempt to Update to larger size
	as.UpdateAddressSpace(expectedSize)
	if as.(*space).size != expectedSize {
		t.Errorf("Update failed to set the new size."+
			"\nexpected: %d\nreceived: %d", expectedSize, as.(*space).size)
	}

	// Attempt to Update to smaller size
	as.UpdateAddressSpace(expectedSize - 1)
	if as.(*space).size != expectedSize {
		t.Errorf("Update failed to set the new size."+
			"\nexpected: %d\nreceived: %d", expectedSize, as.(*space).size)
	}
}

// Tests that Space.UpdateAddressSpace sends the new size to all registered
// channels.
func TestSpace_UpdateAddressSpace_GetAndChannels(t *testing.T) {
	as := NewAddressSpace(initSize)
	var wg sync.WaitGroup
	expectedSize := uint8(42)

	// Start threads that are waiting for an Update
	wait := []chan uint8{make(chan uint8), make(chan uint8), make(chan uint8)}
	for _, waitChan := range wait {
		go func(waitChan chan uint8) {
			waitChan <- as.GetAddressSpace()
		}(waitChan)
	}

	// Wait on threads
	for i, waitChan := range wait {
		wg.Add(1)
		go func(i int, waitChan chan uint8) {
			defer wg.Done()

			select {
			case size := <-waitChan:
				if size != expectedSize {
					t.Errorf("Thread %d received unexpected size."+
						"\nexpected: %d\nreceived: %d", i, expectedSize, size)
				}
			case <-time.After(25 * time.Millisecond):
				t.Errorf("Timed out waiting for get to return on thread %d.", i)
			}
		}(i, waitChan)
	}

	// Register channels
	notifyChannels := make(map[string]chan uint8)
	var notifyChan chan uint8
	var err error
	var chanID string
	for i := 0; i < 3; i++ {
		chanID = strconv.Itoa(i)
		notifyChannels[chanID], err = as.RegisterAddressSpaceNotification(chanID)
		if err != nil {
			t.Errorf("Failed to reigster channel: %+v", err)
		}
	}

	// Wait for new size on channels
	for chanID, notifyChan := range notifyChannels {
		wg.Add(1)
		go func(chanID string, notifyChan chan uint8) {
			defer wg.Done()

			select {
			case size := <-notifyChan:
				t.Errorf("Received size %d on channel %s when it should not "+
					"have.", size, chanID)
			case <-time.After(20 * time.Millisecond):
			}
		}(chanID, notifyChan)
	}

	time.Sleep(5 * time.Millisecond)

	// Attempt to Update to larger size
	as.UpdateAddressSpace(expectedSize)

	wg.Wait()

	// Unregistered one channel and make sure it will not receive
	delete(notifyChannels, chanID)
	as.UnregisterAddressSpaceNotification(chanID)

	expectedSize++

	// Wait for new size on channels
	for chanID, notifyChan := range notifyChannels {
		wg.Add(1)
		go func(chanID string, notifyChan chan uint8) {
			defer wg.Done()

			select {
			case size := <-notifyChan:
				if size != expectedSize {
					t.Errorf("Failed to receive expected size on channel %s."+
						"\nexpected: %d\nreceived: %d",
						chanID, expectedSize, size)
				}
			case <-time.After(20 * time.Millisecond):
				t.Errorf("Timed out waiting on channel %s", chanID)
			}
		}(chanID, notifyChan)
	}

	// Wait for timeout on unregistered channel
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case size := <-notifyChan:
			t.Errorf("Received size %d on channel %s when it should not have.",
				size, chanID)
		case <-time.NewTimer(20 * time.Millisecond).C:
		}
	}()

	time.Sleep(5 * time.Millisecond)

	// Attempt to Update to larger size
	as.UpdateAddressSpace(expectedSize)

	wg.Wait()
}

// Tests that a channel created by Space.RegisterAddressSpaceNotification
// receives the expected size when triggered.
func TestSpace_RegisterAddressSpaceNotification(t *testing.T) {
	as := NewAddressSpace(initSize)
	expectedSize := uint8(42)

	// Register channel
	chanID := "chanID"
	sizeChan, err := as.RegisterAddressSpaceNotification(chanID)
	if err != nil {
		t.Errorf("RegisterNotification returned an error: %+v", err)
	}

	// Wait on channel or error after timing out
	go func() {
		select {
		case size := <-sizeChan:
			if size != expectedSize {
				t.Errorf("received wrong size on channel."+
					"\nexpected: %d\nreceived: %d", expectedSize, size)
			}
		case <-time.After(10 * time.Millisecond):
			t.Error("Timed out waiting on channel.")
		}
	}()

	// Send on channel
	select {
	case as.(*space).notifyMap[chanID] <- expectedSize:
	default:
		t.Errorf("Sent on channel %s that should not be in map.", chanID)
	}
}

// Tests that when Space.UnregisterAddressSpaceNotification unregisters a
// channel and that it no longer can be triggered from the map.
func TestSpace_UnregisterAddressSpaceNotification(t *testing.T) {
	as := NewAddressSpace(initSize)
	expectedSize := uint8(42)

	// Register channel and then unregister it
	chanID := "chanID"
	sizeChan, err := as.RegisterAddressSpaceNotification(chanID)
	if err != nil {
		t.Errorf("RegisterNotification returned an error: %+v", err)
	}
	as.UnregisterAddressSpaceNotification(chanID)

	// Wait for timeout or error if the channel receives
	go func() {
		select {
		case size := <-sizeChan:
			t.Errorf("Received %d on channel %s that should not be in map.",
				size, chanID)
		case <-time.NewTimer(10 * time.Millisecond).C:
		}
	}()

	// Send on channel
	select {
	case as.(*space).notifyMap[chanID] <- expectedSize:
		t.Errorf("Sent size %d on channel %s that should not be in map.",
			expectedSize, chanID)
	default:
	}
}
