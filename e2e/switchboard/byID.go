///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package switchboard

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/xx_network/primitives/id"
)

type byId struct {
	list    map[id.ID]*set.Set
	generic *set.Set
}

// builds a new byID structure
// registers an empty ID and the designated zero ID as generic
func newById() *byId {
	bi := &byId{
		list:    make(map[id.ID]*set.Set),
		generic: set.New(),
	}

	//make the zero IDs, which are defined as any, all point to the generic
	bi.list[*AnyUser()] = bi.generic
	bi.list[id.ID{}] = bi.generic

	return bi
}

// returns a set associated with the passed ID unioned with the generic return
func (bi *byId) Get(uid *id.ID) *set.Set {
	lookup, ok := bi.list[*uid]
	if !ok {
		return bi.generic
	} else {
		return lookup.Union(bi.generic)
	}
}

// adds a listener to a set for the given ID. Creates a new set to add it to if
// the set does not exist
func (bi *byId) Add(uid *id.ID, l Listener) *set.Set {
	s, ok := bi.list[*uid]
	if !ok {
		s = set.New(l)
		bi.list[*uid] = s
	} else {
		s.Insert(l)
	}

	return s
}

// Removes the passed listener from the set for UserID and
// deletes the set if it is empty if the ID is not a generic one
func (bi *byId) Remove(uid *id.ID, l Listener) {
	s, ok := bi.list[*uid]
	if ok {
		s.Remove(l)

		if s.Len() == 0 && !uid.Cmp(AnyUser()) && !uid.Cmp(&id.ID{}) {
			delete(bi.list, *uid)
		}
	}
}
