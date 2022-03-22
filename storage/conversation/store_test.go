///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// Read jww trace output to determine if key names are ok
func TestStore_Get_Prefix(t *testing.T) {
	// Uncomment to print keys that Set and get are called on
	jww.SetStdoutThreshold(jww.LevelTrace)

	// It's a conversation with a partner, so does there need to be an additional layer of hierarchy here later?
	rootKv := versioned.NewKV(make(ekv.Memstore))
	store := NewStore(rootKv)
	conv := store.Get(id.NewIdFromUInt(8, id.User, t))
	t.Log(conv)
}

// Happy path.
func TestStore_Delete(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	store := NewStore(kv)
	pids := make([]*id.ID, 10)

	// Generate list of IDs
	for i := range pids {
		pids[i] = id.NewIdFromUInt(rand.Uint64(), id.User, t)
	}

	// Add IDs to storage and memory
	for _, pid := range pids {
		store.Get(pid)
	}

	// delete conversations with IDs with even numbered indexes
	for i := 0; i < len(pids); i += 2 {
		store.Delete(pids[i])
	}

	// Ensure even numbered conversation were deleted and all others still exist
	for i, pid := range pids {
		_, exists := store.loadedConversations[*pid]
		if i%2 == 0 {
			if exists {
				t.Errorf("%d. delete() failed to delete the conversation "+
					"(ID %s) from memory.", i, pid)
			}
		} else if !exists {
			t.Errorf("%d. delete() unexpetedly deleted the conversation "+
				"(ID %s) from memory.", i, pid)
		}
	}
}
