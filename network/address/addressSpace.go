package address

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
)

const (
	// The initial value for the address space size. This value signifies that
	// the address space size has not yet been updated.
	initSize = 1
)

// Space contains the current address space size used for creating address IDs
// and the infrastructure to alert other processes when an update occurs.
type Space interface {
	GetAddressSpace() uint8
	GetAddressSpaceWithoutWait() uint8
	UpdateAddressSpace(newSize uint8)
	RegisterAddressSpaceNotification(tag string) (chan uint8, error)
	UnregisterAddressSpaceNotification(tag string)
}

type space struct {
	size      uint8
	notifyMap map[string]chan uint8
	cond      *sync.Cond
}

// NewAddressSpace initialises a new Space and returns it.
func NewAddressSpace() Space {
	return &space{
		size:      initSize,
		notifyMap: make(map[string]chan uint8),
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

// GetAddressSpace returns the current address space size. It blocks until an
// address space size is set.
func (as *space) GetAddressSpace() uint8 {
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

// GetAddressSpaceWithoutWait returns the current address space size regardless
// if it has been set yet.
func (as *space) GetAddressSpaceWithoutWait() uint8 {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()
	return as.size
}

// UpdateAddressSpace updates the address space size to the new size, if it is
// larger. Then, each registered channel is notified of the Update. If this was
// the first time that the address space size was set, then the conditional
// broadcasts to stop blocking for all threads waiting on GetAddressSpace.
func (as *space) UpdateAddressSpace(newSize uint8) {
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
				jww.ERROR.Printf("Failed to send address space Update of %d "+
					"on channel with ID %s", as.size, chanID)
			}
		}
	}
}

// RegisterAddressSpaceNotification returns a channel that will trigger for
// every address space size update. The provided tag is the unique ID for the
// channel. Returns an error if the tag is already used.
func (as *space) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	if _, exists := as.notifyMap[tag]; exists {
		return nil, errors.Errorf("tag %q already exists in notify map", tag)
	}

	as.notifyMap[tag] = make(chan uint8, 1)

	return as.notifyMap[tag], nil
}

// UnregisterAddressSpaceNotification stops broadcasting address space size
// updates on the channel with the specified tag.
func (as *space) UnregisterAddressSpaceNotification(tag string) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	delete(as.notifyMap, tag)
}
