package conversation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Read jww trace output to determine if key names are ok
func TestStore_Get_Prefix(t *testing.T) {
	// Uncomment to print keys that Set and Get are called on
	jww.SetStdoutThreshold(jww.LevelTrace)

	// It's a conversation with a partner, so does there need to be an additional layer of hierarchy here later?
	rootKv := versioned.NewKV(make(ekv.Memstore))
	store := NewStore(rootKv)
	conv := store.Get(id.NewIdFromUInt(8, id.User, t))
	t.Log(conv)
}
