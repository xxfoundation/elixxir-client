package ud

import (
	"encoding/binary"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const isRegisteredKey = "isRegisteredKey"
const isRegisteredVersion = 0

// IsRegistered loads from storage if the user is registered with user
// discovery.
func IsRegistered(kv *versioned.KV) bool {
	_, err := kv.Get(isRegisteredKey, isRegisteredVersion)
	if err != nil {
		return false
	}

	return true
}

// setRegistered sets the user to registered
func setRegistered(kv *versioned.KV) error {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, 1)
	obj := &versioned.Object{
		Version:   isRegisteredVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	if err := kv.Set(isRegisteredKey, obj); err != nil {
		jww.FATAL.Panicf("Failed to store that the client is "+
			"registered: %+v", err)
	}
	return nil
}
