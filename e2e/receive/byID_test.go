////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// tests the newByID functions forms a properly constructed byId
func TestById_newById(t *testing.T) {
	nbi := newById()

	if nbi.list == nil {
		t.Errorf("No list created")
	}

	if nbi.generic == nil {
		t.Errorf("No generic created")
	}

	if nbi.generic != nbi.list[id.ZeroUser] {
		t.Errorf("zero user not registered as generic")
	}

	if nbi.generic != nbi.list[id.ID{}] {
		t.Errorf("zero id not registered as generic")
	}
}

// tests that when nothing has been added an empty set is returned
func TestById_Get_Empty(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	s := nbi.Get(uid)

	if s.Len() != 0 {
		t.Errorf("Should not have returned a set")
	}
}

// tests that getting a set for a specific ID returns that set
func TestById_Get_Selected(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	set1 := set.New(0)

	nbi.list[*uid] = set1

	s := nbi.Get(uid)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	if !s.SubsetOf(set1) || !set1.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

// tests that when getting a specific ID which is not there returns the generic
// set if is present
func TestById_Get_Generic(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	nbi.generic.Insert(0)

	s := nbi.Get(uid)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	if !s.SubsetOf(nbi.generic) || !nbi.generic.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

// tests that when getting a specific ID is there and there are elements
// in the generic that the union of the two is returned
func TestById_Get_GenericSelected(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	set1 := set.New(0)

	nbi.list[*uid] = set1

	nbi.generic.Insert(1)

	s := nbi.Get(uid)

	if s.Len() == 0 {
		t.Errorf("Should have returned a set")
	}

	setUnion := set1.Union(nbi.generic)

	if !s.SubsetOf(setUnion) || !setUnion.SubsetOf(s) {
		t.Errorf("Wrong set returned")
	}
}

// Tests that when adding to a set which does not exist, the set is created
func TestById_Add_New(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	lid := ListenerID{uid, 1, &funcListener{}}

	nbi.Add(lid)

	s := nbi.list[*uid]

	if s.Len() != 1 {
		t.Errorf("Should a set of the wrong size")
	}

	if !s.Has(lid) {
		t.Errorf("Wrong set returned")
	}
}

// Tests that when adding to a set which does exist, the set is retained and
// added to
func TestById_Add_Old(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	lid1 := ListenerID{uid, 1, &funcListener{}}
	lid2 := ListenerID{uid, 1, &funcListener{}}

	set1 := set.New(lid1)

	nbi.list[*uid] = set1

	nbi.Add(lid2)

	s := nbi.list[*uid]

	if s.Len() != 2 {
		t.Errorf("Incorrect set length.\nexpected: %d\nreceived: %d", 2, s.Len())
	}

	if !s.Has(lid1) {
		t.Errorf("Set does not include the initial listener")
	}

	if !s.Has(lid2) {
		t.Errorf("Set does not include the new listener")
	}
}

// Tests that when adding to a generic ID, the listener is added to the
// generic set
func TestById_Add_Generic(t *testing.T) {
	nbi := newById()

	lid1 := ListenerID{&id.ID{}, 1, &funcListener{}}
	lid2 := ListenerID{AnyUser(), 1, &funcListener{}}

	nbi.Add(lid1)
	nbi.Add(lid2)

	s := nbi.generic

	if s.Len() != 2 {
		t.Errorf("Should have returned a set of size 2")
	}

	if !s.Has(lid1) {
		t.Errorf("Set does not include the ZeroUser listener")
	}

	if !s.Has(lid2) {
		t.Errorf("Set does not include the empty user listener")
	}
}

// Tests that removing a listener from a set with multiple listeners removes the
// listener but maintains the set
func TestById_Remove_ManyInSet(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	lid1 := ListenerID{uid, 1, &funcListener{}}
	lid2 := ListenerID{uid, 1, &funcListener{}}

	set1 := set.New(lid1, lid2)

	nbi.list[*uid] = set1

	nbi.Remove(lid1)

	if _, ok := nbi.list[*uid]; !ok {
		t.Errorf("Set removed when it should not have been")
	}

	if set1.Len() != 1 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(lid1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}

	if !set1.Has(lid2) {
		t.Errorf("Listener 2 not still in set, it should be")
	}

}

// Tests that removing a listener from a set with a single listener removes the
// listener and the set
func TestById_Remove_SingleInSet(t *testing.T) {
	nbi := newById()

	uid := id.NewIdFromUInt(42, id.User, t)

	lid1 := ListenerID{uid, 1, &funcListener{}}

	set1 := set.New(lid1)

	nbi.list[*uid] = set1

	nbi.Remove(lid1)

	if _, ok := nbi.list[*uid]; ok {
		t.Errorf("Set not removed when it should have been")
	}

	if set1.Len() != 0 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(lid1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}
}

// Tests that removing a listener from a set with a single listener removes the
// listener and not the set when the ID iz ZeroUser
func TestById_Remove_SingleInSet_ZeroUser(t *testing.T) {
	nbi := newById()

	uid := &id.ZeroUser

	lid1 := ListenerID{uid, 1, &funcListener{}}

	set1 := set.New(lid1)

	nbi.list[*uid] = set1

	nbi.Remove(lid1)

	if _, ok := nbi.list[*uid]; !ok {
		t.Errorf("Set removed when it should not have been")
	}

	if set1.Len() != 0 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(lid1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}
}

// Tests that removing a listener from a set with a single listener removes the
// listener and not the set when the ID iz ZeroUser
func TestById_Remove_SingleInSet_EmptyUser(t *testing.T) {
	nbi := newById()

	uid := &id.ID{}

	lid1 := ListenerID{uid, 1, &funcListener{}}

	set1 := set.New(lid1)

	nbi.list[*uid] = set1

	nbi.Remove(lid1)

	if _, ok := nbi.list[*uid]; !ok {
		t.Errorf("Set removed when it should not have been")
	}

	if set1.Len() != 0 {
		t.Errorf("Set is incorrect length after the remove call: %v",
			set1.Len())
	}

	if set1.Has(lid1) {
		t.Errorf("Listener 1 still in set, it should not be")
	}
}
