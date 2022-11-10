////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// Happy path.
func TestStore_Delete(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	store := NewStore(kv)
	pIDs := make([]*id.ID, 10)

	// Generate list of IDs
	for i := range pIDs {
		pIDs[i] = id.NewIdFromUInt(rand.Uint64(), id.User, t)
	}

	// Add IDs to storage and memory
	for _, pid := range pIDs {
		store.Get(pid)
	}

	// Delete conversations with IDs with even numbered indexes
	for i := 0; i < len(pIDs); i += 2 {
		store.Delete(pIDs[i])
	}

	// Ensure even numbered conversation were deleted and all others still exist
	for i, pid := range pIDs {
		_, exists := store.loadedConversations[*pid]
		if i%2 == 0 {
			if exists {
				t.Errorf("Delete failed to delete the conversation for ID "+
					"%s (%d).", pid, i)
			}
		} else if !exists {
			t.Errorf("Delete unexpetedly deletde the conversation for ID "+
				"%s (%d).", pid, i)
		}
	}
}
