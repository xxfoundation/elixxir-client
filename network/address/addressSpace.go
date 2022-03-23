package address

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
	"testing"
)

const (
	// The initial value for the address space size. This value signifies that
	// the address space size has not yet been updated.
	initSize = 1
)

// Space contains the current address space size used for creating
// address IDs and the infrastructure to alert other processes when an Update
// occurs.
type Space struct {
	size      uint8
	notifyMap map[string]chan uint8
	cond      *sync.Cond
}

// NewAddressSpace initialises a new AddressSpace and returns it.
func NewAddressSpace() *Space {
	return &Space{
		size:      initSize,
		notifyMap: make(map[string]chan uint8),
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

// Get returns the current address space size. It blocks until an address space
// size is set.
func (as *Space) Get() uint8 {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	// If the size has been set, then return the current size
	if as.size != initSize {
		return as.size
	}

	// If the size is not set, then block until it is set
	as.cond.Wait()

	return as.size
}

// GetWithoutWait returns the current address space size regardless if it has
// been set yet.
func (as *Space) GetWithoutWait() uint8 {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()
	return as.size
}

// Update updates the address space size to the new size, if it is larger. Then,
// each registered channel is notified of the Update. If this was the first time
// that the address space size was set, then the conditional broadcasts to stop
// blocking for all threads waiting on Get.
func (as *Space) Update(newSize uint8) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	// Skip Update if the address space size is unchanged
	if as.size >= newSize {
		return
	}

	// Update address space size
	oldSize := as.size
	as.size = newSize
	jww.INFO.Printf("Updated address space size from %d to %d", oldSize, as.size)

	// Broadcast that the address space size is set, if set for the first time
	if oldSize == initSize {
		as.cond.Broadcast()
	} else {
		// Broadcast the new address space size to all registered channels
		for chanID, sizeChan := range as.notifyMap {
			select {
			case sizeChan <- as.size:
			default:
				jww.ERROR.Printf("Failed to send address space Update of %d on "+
					"channel with ID %s", as.size, chanID)
			}
		}
	}
}

// RegisterNotification returns a channel that will trigger for every address
// space size Update. The provided tag is the unique ID for the channel.
// Returns an error if the tag is already used.
func (as *Space) RegisterNotification(tag string) (chan uint8, error) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	if _, exists := as.notifyMap[tag]; exists {
		return nil, errors.Errorf("tag \"%s\" already exists in notify map", tag)
	}

	as.notifyMap[tag] = make(chan uint8, 1)

	return as.notifyMap[tag], nil
}

// UnregisterNotification stops broadcasting address space size updates on the
// channel with the specified tag.
func (as *Space) UnregisterNotification(tag string) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	delete(as.notifyMap, tag)
}

// NewTestAddressSpace initialises a new AddressSpace for testing with the given
// size.
func NewTestAddressSpace(newSize uint8, x interface{}) *Space {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("NewTestAddressSpace is restricted to testing only. "+
			"Got %T", x)
	}

	as := &Space{
		size:      initSize,
		notifyMap: make(map[string]chan uint8),
		cond:      sync.NewCond(&sync.Mutex{}),
	}

	as.Update(newSize)

	return as
}
