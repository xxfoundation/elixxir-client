package ud

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const isRegisteredKey = "isRegisteredKey"
const isRegisteredVersion = 0

// isRegistered loads from storage if the client is registered with user
// discovery.
func (m *Manager) isRegistered() bool {
	obj, err := m.kv.Get(isRegisteredKey, isRegisteredVersion)
	if err != nil {
		return false
	}

	return true
}

// isRegistered returns if the client is registered with user discovery
func (m *Manager) setRegistered() error {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 1)
	obj := &versioned.Object{
		Version:   isRegisteredVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	if err := m.kv.Set(isRegisteredKey, isRegisteredVersion, obj); err != nil {
		jww.FATAL.Panicf("Failed to store that the client is "+
			"registered: %+v", err)
	}
	return nil
}
