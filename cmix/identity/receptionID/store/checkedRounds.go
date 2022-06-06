package store

import (
	"container/list"
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
)

const itemsPerBlock = 50

type CheckedRounds struct {
	// Map of round IDs for quick lookup if an ID is stored
	m map[id.Round]interface{}

	// List of round IDs in order of age; oldest in front and newest in back
	l *list.List

	// List of recently added round IDs that need to be stored
	recent []id.Round

	// Saves round IDs in blocks to storage
	store *utility.BlockStore

	// The maximum number of round IDs to store before pruning the oldest
	maxRounds int
}

// NewCheckedRounds returns a new CheckedRounds with an initialized map.
func NewCheckedRounds(maxRounds int, kv *versioned.KV) (*CheckedRounds, error) {
	// Calculate the number of blocks of size itemsPerBlock are needed to store
	// numRoundsToKeep number of round IDs
	numBlocks := maxRounds / itemsPerBlock
	if maxRounds%itemsPerBlock != 0 {
		numBlocks++
	}

	// Create a new BlockStore for storing the round IDs to storage
	store, err := utility.NewBlockStore(itemsPerBlock, numBlocks, kv)
	if err != nil {
		return nil, errors.Errorf(
			"failed to save new checked rounds to storage: %+v", err)
	}

	// Create new CheckedRounds
	return newCheckedRounds(maxRounds, store), nil
}

// newCheckedRounds initialises the lists in CheckedRounds.
func newCheckedRounds(maxRounds int, store *utility.BlockStore) *CheckedRounds {
	return &CheckedRounds{
		m:         make(map[id.Round]interface{}),
		l:         list.New(),
		recent:    make([]id.Round, 0, maxRounds),
		store:     store,
		maxRounds: maxRounds,
	}
}

// LoadCheckedRounds restores the list from storage.
func LoadCheckedRounds(maxRounds int, kv *versioned.KV) (*CheckedRounds, error) {
	// get rounds from storage
	store, rounds, err := utility.LoadBlockStore(kv)
	if err != nil {
		return nil, errors.Errorf(
			"failed to load CheckedRounds from storage: %+v", err)
	}

	// Create new CheckedRounds
	cr := newCheckedRounds(maxRounds, store)

	// Unmarshal round ID byte list into the new CheckedRounds
	cr.unmarshal(rounds)

	return cr, nil
}

// SaveCheckedRounds stores the list to storage.
func (cr *CheckedRounds) SaveCheckedRounds() error {

	// Save to disk
	err := cr.store.Store(cr)
	if err != nil {
		return errors.Errorf("failed to store recent CheckedRounds: %+v", err)
	}

	// Save to disk
	return nil
}

// Next pops the oldest recent round ID from the list and returns it as bytes.
// Returns false if the list is empty
func (cr *CheckedRounds) Next() ([]byte, bool) {
	if len(cr.recent) > 0 {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(cr.recent[0]))
		cr.recent = cr.recent[1:]
		return b, true
	}

	return nil, false
}

// Check determines if the round ID has been added to the checklist. If it has
// not, then it is added and the function returns true. Otherwise, if it already
// exists, then the function returns false.
func (cr *CheckedRounds) Check(rid id.Round) bool {
	// Add the round ID to the checklist if it does not exist and return true
	if _, exists := cr.m[rid]; !exists {
		cr.m[rid] = nil    // Add ID to the map
		cr.l.PushBack(rid) // Add ID to the end of the list
		cr.recent = append(cr.recent, rid)

		// The commented out code below works the same as the Prune function but
		// occurs when adding a round ID to the list. It was decided to use
		// Prune instead so that it does not block even though the savings are
		// probably negligible.
		// // Remove the oldest round ID the list is full
		// if cr.l.Len() > cr.maxRounds {
		// 	oldestID := cr.l.Remove(cr.l.Front()) // Remove oldest from list
		// 	delete(cr.m, oldestID.(id.Round))     // Remove oldest from map
		// }

		return true
	}

	return false
}

// IsChecked determines if the round has been added to the checklist.
func (cr *CheckedRounds) IsChecked(rid id.Round) bool {
	_, exists := cr.m[rid]
	return exists
}

// Prune any rounds that are earlier than the earliestAllowed.
func (cr *CheckedRounds) Prune() {
	if len(cr.m) < cr.maxRounds {
		return
	}
	earliestAllowed := cr.l.Back().Value.(id.Round) - id.Round(cr.maxRounds) + 1

	// Iterate over all the round IDs and remove any that are too old
	for e := cr.l.Front(); e != nil; {
		if e.Value.(id.Round) < earliestAllowed {
			delete(cr.m, e.Value.(id.Round))
			lastE := e
			e = e.Next()
			cr.l.Remove(lastE)
		} else {
			break
		}
	}
}

// unmarshal unmarshalls the list of byte slices into the CheckedRounds map and
// list.
func (cr *CheckedRounds) unmarshal(rounds [][]byte) {
	for _, round := range rounds {
		rid := id.Round(binary.LittleEndian.Uint64(round))
		cr.m[rid] = nil
		cr.l.PushBack(rid)
	}
}
