package sync

import (
	"encoding/json"
	"time"
)

// Patch is the structure which stores both local and remote patches,
// which are sets of commands to update the local KV
// There is one command per key, which is the latest command from the
// device which originated the patch
// they are ordered by timestamp, so they can be quickly iterated over to
// determine which mutations have been processed by a given receiver.
type Patch struct {
	keys map[string]*Mutate
}

func newPatch() *Patch {
	return &Patch{keys: make(map[string]*Mutate)}
}

// AddUnsafe adds a given mutation to the Patch.
// This must only be called on the creator of the patch
// Only call within the transaction log
func (p *Patch) AddUnsafe(key string, m *Mutate) {
	p.keys[key] = m
}

func (p *Patch) Serialize() ([]byte, error) {
	return json.Marshal(&p.keys)
}

func (p *Patch) Deserialize(b []byte) error {
	return json.Unmarshal(b, &p.keys)
}

func (p *Patch) get(key string) (*Mutate, bool) {
	m, exists := p.keys[key]
	return m, exists
}

// Diff combines multiple patches into a single set of updates which can be
// applied to the local KV. It will not process mutations from each patch
// which is not newer thant he lastSeen for that patch unless the key
// that mutation edits was edited by another patch.
// The patches need to be ordered in ascending order by supremacy, defined
// as having larger device IDs. This supremacy will determine which mutation
// is applied in the event they have the same timestamp
// O(2*numPatches*numMutations)
// Diff does not check the _____ for updates because they should already be
// applied
func (p *Patch) Diff(patches []*Patch, lastSeen []time.Time) (
	map[string]*Mutate, []time.Time) {
	mutatedKeys, newLastSeen := p.findKeysWithUpdates(patches, lastSeen)
	return buildMerge(patches, mutatedKeys), newLastSeen
}

func (p *Patch) findKeysWithUpdates(remotePatches []*Patch, lastSeen []time.Time) (map[string]struct{}, []time.Time) {

	// make large to avoid reallocation
	keys := make(map[string]struct{}, 1000)

	newLastSeen := make([]time.Time, len(remotePatches))

	// iterate through all patches except yours
	for idx, patch := range remotePatches {
		if patch == p {
			continue
		}
		last := lastSeen[idx].UnixNano()
		newLast := last
		for key, m := range patch.keys {
			if !(m.Timestamp > last) {
				keys[key] = struct{}{}
				if m.Timestamp > newLast {
					newLast = m.Timestamp
				}
			}
		}
		newLastSeen[idx] = time.Unix(0, newLast).UTC()
	}

	return keys, newLastSeen
}

func buildMerge(patches []*Patch, mutatedKeys map[string]struct{}) map[string]*Mutate {
	output := make(map[string]*Mutate, len(mutatedKeys))

	for key := range mutatedKeys {
		defending := &Mutate{
			Timestamp: 0,
		}

		for _, patch := range patches {
			if contender, exists := patch.get(key); exists {
				// checks if this one is newer or the exact same age
				// implements supremely, if they are the same the one
				// which has the higher device ID, which will be
				// closer to the end of the list, will be skipped
				if !(defending.Timestamp > contender.Timestamp) {
					defending = contender
				}
			}
		}
		output[key] = defending

	}
	return output
}
