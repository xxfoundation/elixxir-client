////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/client/v5/catalog"
)

type byType struct {
	list    map[catalog.MessageType]*set.Set
	generic *set.Set
}

// builds a new byType structure
// registers an AnyType as generic
func newByType() *byType {
	bt := &byType{
		list:    make(map[catalog.MessageType]*set.Set),
		generic: set.New(),
	}

	// make the zero messages, which are defined as AnyType,
	// point to the generic
	bt.list[AnyType] = bt.generic

	return bt
}

// returns a set associated with the passed messageType unioned with the
// generic return
func (bt *byType) Get(messageType catalog.MessageType) *set.Set {
	lookup, ok := bt.list[messageType]
	if !ok {
		return bt.generic
	} else {
		return lookup.Union(bt.generic)
	}
}

// adds a listener to a set for the given messageType. Creates a new set to add
// it to if the set does not exist
func (bt *byType) Add(lid ListenerID) *set.Set {
	s, ok := bt.list[lid.messageType]
	if !ok {
		s = set.New(lid)
		bt.list[lid.messageType] = s
	} else {
		s.Insert(lid)
	}

	return s
}

// Removes the passed listener from the set for messageType and
// deletes the set if it is empty and the type is not AnyType
func (bt *byType) Remove(lid ListenerID) {
	s, ok := bt.list[lid.messageType]
	if ok {
		s.Remove(lid)

		if s.Len() == 0 && lid.messageType != AnyType {
			delete(bt.list, lid.messageType)
		}
	}
}
