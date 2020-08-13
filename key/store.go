package key

import (
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type Store struct {
	managers         map[id.ID]*Manager
	fingerprintToKey map[format.Fingerprint]*Key
}

type StoreDisk struct {
	contacts []id.ID
}

func (s *Store) CleanManager(partner id.ID) error {
	//lookup

	//get sessions to be removed

}
