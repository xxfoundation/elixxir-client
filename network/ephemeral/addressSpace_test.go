package ephemeral

import (
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

// Unit test of NewAddressSpace.
func Test_newAddressSpace(t *testing.T) {
	expected := &AddressSpace{
		size:      initSize,
		notifyMap: make(map[string]chan uint8),
		cond:      sync.NewCond(&sync.Mutex{}),
	}

	as := NewAddressSpace()

	if !reflect.DeepEqual(expected, as) {
		t.Errorf("NewAddressSpace failed to return the expected AddressSpace."+
			"\nexpected: %+v\nreceived: %+v", expected, as)
	}
}

// Test that AddressSpace.Get blocks when the address space size has not been
// set and that it does not block when it has been set.
func Test_addressSpace_Get(t *testing.T) {
	as := NewAddressSpace()
	expectedSize := uint8(42)

	// Call get and error if it does not block
	wait := make(chan uint8)
	go func() { wait <- as.Get() }()
	select {
	case size := <-wait:
		t.Errorf("get failed to block and returned size %d.", size)
	case <-time.NewTimer(10 * time.Millisecond).C:
	}

	// Update address size
	as.cond.L.Lock()
	as.size = expectedSize
	as.cond.L.Unlock()

	// Call get and error if it does block
	wait = make(chan uint8)
	go func() { wait <- as.Get() }()
	select {
	case size := <-wait:
		if size != expectedSize {
			t.Errorf("get returned the wrong size.\nexpected: %d\nreceived: %d",
				expectedSize, size)
		}
	case <-time.NewTimer(15 * time.Millisecond).C:
		t.Error("get blocking when the size has been updated.")
	}
}

// Test that AddressSpace.Get blocks until the condition broadcasts.
func Test_addressSpace_Get_WaitBroadcast(t *testing.T) {
	as := NewAddressSpace()

	wait := make(chan uint8)
	go func() { wait <- as.Get() }()

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

	as.cond.Broadcast()
}

// Unit test of AddressSpace.GetWithoutWait.
func Test_addressSpace_GetWithoutWait(t *testing.T) {
	as := NewAddressSpace()

	size := as.GetWithoutWait()
	if size != initSize {
		t.Errorf("GetWithoutWait returned the wrong size."+
			"\nexpected: %d\nreceived: %d", initSize, size)
	}
}

// Tests that AddressSpace.Update only updates the size when it is larger.
func Test_addressSpace_update(t *testing.T) {
	as := NewAddressSpace()
	expectedSize := uint8(42)

	// Attempt to Update to larger size
	as.Update(expectedSize)
	if as.size != expectedSize {
		t.Errorf("Update failed to set the new size."+
			"\nexpected: %d\nreceived: %d", expectedSize, as.size)
	}

	// Attempt to Update to smaller size
	as.Update(expectedSize - 1)
	if as.size != expectedSize {
		t.Errorf("Update failed to set the new size."+
			"\nexpected: %d\nreceived: %d", expectedSize, as.size)
	}
}

// Tests that AddressSpace.Update sends the new size to all registered channels.
func Test_addressSpace_update_GetAndChannels(t *testing.T) {
	as := NewAddressSpace()
	var wg sync.WaitGroup
	expectedSize := uint8(42)

	// Start threads that are waiting for an Update
	wait := []chan uint8{make(chan uint8), make(chan uint8), make(chan uint8)}
	for _, waitChan := range wait {
		go func(waitChan chan uint8) { waitChan <- as.Get() }(waitChan)
	}

	// Wait on threads
	for i, waitChan := range wait {
		go func(i int, waitChan chan uint8) {
			wg.Add(1)
			defer wg.Done()

			select {
			case size := <-waitChan:
				if size != expectedSize {
					t.Errorf("Thread %d received unexpected size."+
						"\nexpected: %d\nreceived: %d", i, expectedSize, size)
				}
			case <-time.NewTimer(20 * time.Millisecond).C:
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
		notifyChannels[chanID], err = as.RegisterNotification(chanID)
		if err != nil {
			t.Errorf("Failed to regisdter channel: %+v", err)
		}
	}

	// Wait for new size on channels
	for chanID, notifyChan := range notifyChannels {
		go func(chanID string, notifyChan chan uint8) {
			wg.Add(1)
			defer wg.Done()

			select {
			case size := <-notifyChan:
				t.Errorf("Received size %d on channel %s when it should not have.",
					size, chanID)
			case <-time.NewTimer(20 * time.Millisecond).C:
			}
		}(chanID, notifyChan)
	}

	time.Sleep(5 * time.Millisecond)

	// Attempt to Update to larger size
	as.Update(expectedSize)

	wg.Wait()

	// Unregistered one channel and make sure it will not receive
	delete(notifyChannels, chanID)
	as.UnregisterNotification(chanID)

	expectedSize++

	// Wait for new size on channels
	for chanID, notifyChan := range notifyChannels {
		go func(chanID string, notifyChan chan uint8) {
			wg.Add(1)
			defer wg.Done()

			select {
			case size := <-notifyChan:
				if size != expectedSize {
					t.Errorf("Failed to receive expected size on channel %s."+
						"\nexpected: %d\nreceived: %d", chanID, expectedSize, size)
				}
			case <-time.NewTimer(20 * time.Millisecond).C:
				t.Errorf("Timed out waiting on channel %s", chanID)
			}
		}(chanID, notifyChan)
	}

	// Wait for timeout on unregistered channel
	go func() {
		wg.Add(1)
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
	as.Update(expectedSize)

	wg.Wait()
}

// Tests that a channel created by AddressSpace.RegisterNotification receives
// the expected size when triggered.
func Test_addressSpace_RegisterNotification(t *testing.T) {
	as := NewAddressSpace()
	expectedSize := uint8(42)

	// Register channel
	chanID := "chanID"
	sizeChan, err := as.RegisterNotification(chanID)
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
		case <-time.NewTimer(10 * time.Millisecond).C:
			t.Error("Timed out waiting on channel.")
		}
	}()

	// Send on channel
	select {
	case as.notifyMap[chanID] <- expectedSize:
	default:
		t.Errorf("Sent on channel %s that should not be in map.", chanID)
	}
}

// Tests that when AddressSpace.UnregisterNotification unregisters a channel,
// it no longer can be triggered from the map.
func Test_addressSpace_UnregisterNotification(t *testing.T) {
	as := NewAddressSpace()
	expectedSize := uint8(42)

	// Register channel and then unregister it
	chanID := "chanID"
	sizeChan, err := as.RegisterNotification(chanID)
	if err != nil {
		t.Errorf("RegisterNotification returned an error: %+v", err)
	}
	as.UnregisterNotification(chanID)

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
	case as.notifyMap[chanID] <- expectedSize:
		t.Errorf("Sent size %d on channel %s that should not be in map.",
			expectedSize, chanID)
	default:
	}
}
