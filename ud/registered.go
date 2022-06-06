package ud

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"sync/atomic"
)

const isRegisteredKey = "isRegisteredKey"
const isRegisteredVersion = 0

// loadRegistered loads from storage if the client is registered with user
// discovery.
func (m *Manager) loadRegistered() {
	var isReg = uint32(0)
	obj, err := m.storage.Get(isRegisteredKey)
	if err != nil {
		jww.INFO.Printf("Failed to load is registered, "+
			"assuming un-registered: %s", err)
	} else {
		isReg = binary.BigEndian.Uint32(obj.Data)
	}

	m.registered = &isReg
}

// IsRegistered returns if the client is registered with user discovery
func (m *Manager) IsRegistered() bool {
	return atomic.LoadUint32(m.registered) == 1
}

// setRegistered sets the manager's state to registered.
func (m *Manager) setRegistered() error {
	if !atomic.CompareAndSwapUint32(m.registered, 0, 1) {
		return errors.New("cannot register with User Discovery when " +
			"already registered")
	}

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 1)

	obj := &versioned.Object{
		Version:   isRegisteredVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	if err := m.storage.Set(isRegisteredKey, obj); err != nil {
		jww.FATAL.Panicf("Failed to store that the client is "+
			"registered: %+v", err)
	}
	return nil
}
