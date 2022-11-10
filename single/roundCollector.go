////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

// roundCollector keeps track of a list of unique rounds. Multiple inserts of
// the same round are ignored. It is multi-thread safe.
type roundCollector struct {
	list map[id.Round]rounds.Round
	mux  sync.Mutex
}

// newRoundIdCollector initialises a new roundCollector with a list of the
// given size. Size is not necessary and can be larger or smaller than the real
// size.
func newRoundIdCollector(size int) *roundCollector {
	return &roundCollector{
		list: make(map[id.Round]rounds.Round, size),
	}
}

// add inserts a new round to the list.
func (rc *roundCollector) add(round rounds.Round) {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	rc.list[round.ID] = round
}

// getList returns the list of round IDs.
func (rc *roundCollector) getList() []rounds.Round {
	rc.mux.Lock()
	defer rc.mux.Unlock()

	list := make([]rounds.Round, 0, len(rc.list))

	for _, round := range rc.list {
		list = append(list, round)
	}

	return list
}
