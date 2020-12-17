package ud

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"time"
)

const commIDKey = "commIDKey"
const commIDVersion = 0

// getCommID returns the ID for the next comm. IDs are generated sequentially.
func (m *Manager) getCommID() uint64 {

	m.commIDLock.Lock()
	defer m.commIDLock.Unlock()
	returnedID := m.commID

	// Increment ID for next get
	m.commID++

	// Save ID storage
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, m.commID)

	obj := &versioned.Object{
		Version:   commIDVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	if err := m.storage.Set(commIDKey, obj); err != nil {
		jww.FATAL.Panicf("Failed to store the next commID: %+v", err)
	}

	return returnedID
}

// loadCommID retrieves the next comm ID from storage.
func (m *Manager) loadCommID() {
	m.commIDLock.Lock()
	defer m.commIDLock.Unlock()

	obj, err := m.storage.Get(commIDKey)
	if err != nil {
		jww.WARN.Printf("Failed to get the commID; restarting at zero: %+v", err)
		return
	}

	m.commID = binary.BigEndian.Uint64(obj.Data)
}
