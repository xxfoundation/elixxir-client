package partition

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Read jww trace output to determine if key names are ok
func TestStore_AddFirst_Prefix(t *testing.T) {
	// Uncomment to print keys that Set and Get are called on
	jww.SetStdoutThreshold(jww.LevelTrace)

	rootKv := versioned.NewKV(make(ekv.Memstore))
	store := New(rootKv)
	// Currently fails w/ a panic but shouldn't!
	partner := id.NewIdFromUInt(8, id.User, t)
	const messageID = 1
	store.AddFirst(partner, message.Raw, messageID, 0, 1, time.Now(), []byte("your favorite message"))
}
