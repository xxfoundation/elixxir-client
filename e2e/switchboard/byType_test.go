///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package switchboard

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/client/interfaces/message"
	"testing"
)

func TestByType_newByType(t *testing.T) {
	nbt := newByType()

	if nbt.list == nil {
		t.Errorf("No list created")
	}

	if nbt.generic == nil {
		t.Errorf("No generic created")
	}

	if nbt.generic != nbt.list[0] {
		t.Errorf("zero message type not registered as generic")
	}

}

func TestByType_Get_Empty(t *testing.T) {
	nbt := newByType()

	s := nbt.Get(42)

	if s.Len() != 0 {
		t.Errorf("Should not have returned a set")
	}
}

func TestByType_Get_Selected(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	set1 := set.New(0)

	nbt.list[m] = set1

	s := nbt.Get(m)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	if !s.SubsetOf(set1) || !set1.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

func TestByType_Get_Generic(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	nbt.generic.Insert(0)

	s := nbt.Get(m)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	if !s.SubsetOf(nbt.generic) || !nbt.generic.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

func TestByType_Get_GenericSelected(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	nbt.generic.Insert(1)

	set1 := set.New(0)

	nbt.list[m] = set1

	s := nbt.Get(m)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	setUnion := set1.Union(nbt.generic)

	if !s.SubsetOf(setUnion) || !setUnion.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

// Tests that when adding to a set which does not exist, the set is created
func TestByType_Add_New(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	l := &funcListener{}

	nbt.Add(m, l)

	s := nbt.list[m]

	if s.Len() != 1 {
		t.Errorf("Should a set of the wrong size")
	}

	if !s.Has(l) {
		t.Errorf("Wrong set returned")
	}
}

// Tests that when adding to a set which does exist, the set is retained and
// added to
func TestByType_Add_Old(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	l1 := &funcListener{}
	l2 := &funcListener{}

	set1 := set.New(l1)

	nbt.list[m] = set1

	nbt.Add(m, l2)

	s := nbt.list[m]

	if s.Len() != 2 {
		t.Errorf("Should have returned a set")
	}

	if !s.Has(l1) {
		t.Errorf("Set does not include the initial listener")
	}

	if !s.Has(l2) {
		t.Errorf("Set does not include the new listener")
	}
}

// Tests that when adding to a generic ID, the listener is added to the
// generic set
func TestByType_Add_Generic(t *testing.T) {
	nbt := newByType()

	l1 := &funcListener{}

	nbt.Add(AnyType, l1)

	s := nbt.generic

	if s.Len() != 1 {
		t.Errorf("Should have returned a set of size 2")
	}

	if !s.Has(l1) {
		t.Errorf("Set does not include the ZeroUser listener")
	}
}

// Tests that removing a listener from a set with a single listener removes the
// listener and the set
func TestByType_Remove_SingleInSet(t *testing.T) {
	nbt := newByType()

	m := message.Type(42)

	l1 := &funcListener{}

	set1 := set.New(l1)

	nbt.list[m] = set1

	nbt.Remove(m, l1)

	if _, ok := nbt.list[m]; ok {
		t.Errorf("Set not removed when it should have been")
	}

	if set1.Len() != 0 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(l1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}
}

// Tests that removing a listener from a set with a single listener removes the
// listener and not the set when the ID iz ZeroUser
func TestByType_Remove_SingleInSet_AnyType(t *testing.T) {
	nbt := newByType()

	m := AnyType

	l1 := &funcListener{}

	set1 := set.New(l1)

	nbt.list[m] = set1

	nbt.Remove(m, l1)

	if _, ok := nbt.list[m]; !ok {
		t.Errorf("Set removed when it should not have been")
	}

	if set1.Len() != 0 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(l1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}
}
