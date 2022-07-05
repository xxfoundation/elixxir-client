////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
	"time"
)

// Tests that NewConnectionList returned the expected new ConnectionList.
func TestNewConnectionList(t *testing.T) {
	expected := &ConnectionList{
		list: make(map[id.ID]Connection),
		p:    DefaultConnectionListParams(),
	}

	cl := NewConnectionList(expected.p)

	if !reflect.DeepEqual(expected, cl) {
		t.Errorf("New ConnectionList did not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, cl)
	}
}

// Tests that ConnectionList.Add adds all the given connections to the list.
func TestConnectionList_Add(t *testing.T) {
	cl := NewConnectionList(DefaultConnectionListParams())

	expected := map[id.ID]Connection{
		*id.NewIdFromString("p1", id.User, t): &handler{
			partner: &mockPartner{partnerId: id.NewIdFromString("p1", id.User, t)}},
		*id.NewIdFromString("p2", id.User, t): &handler{
			partner: &mockPartner{partnerId: id.NewIdFromString("p2", id.User, t)}},
		*id.NewIdFromString("p3", id.User, t): &handler{
			partner: &mockPartner{partnerId: id.NewIdFromString("p3", id.User, t)}},
		*id.NewIdFromString("p4", id.User, t): &handler{
			partner: &mockPartner{partnerId: id.NewIdFromString("p4", id.User, t)}},
		*id.NewIdFromString("p5", id.User, t): &handler{
			partner: &mockPartner{partnerId: id.NewIdFromString("p5", id.User, t)}},
	}

	for _, c := range expected {
		cl.Add(c)
	}

	if !reflect.DeepEqual(expected, cl.list) {
		t.Errorf("List does not have expected connections."+
			"\nexpected: %+v\nreceived: %+v", expected, cl.list)
	}

}

// Tests that ConnectionList.Cleanup deletes only stale connections from the
// list and that they are closed.
func TestConnectionList_Cleanup(t *testing.T) {
	cl := NewConnectionList(DefaultConnectionListParams())

	list := []*mockConnection{
		{
			partner: &mockPartner{partnerId: id.NewIdFromString("p0", id.User, t)},
			lastUse: netTime.Now().Add(-(cl.p.MaxAge * 2)),
		}, {
			partner: &mockPartner{partnerId: id.NewIdFromString("p1", id.User, t)},
			lastUse: netTime.Now().Add(-(cl.p.MaxAge / 2)),
		}, {
			partner: &mockPartner{partnerId: id.NewIdFromString("p2", id.User, t)},
			lastUse: netTime.Now().Add(-(cl.p.MaxAge + 10)),
		}, {
			partner: &mockPartner{partnerId: id.NewIdFromString("p3", id.User, t)},
			lastUse: netTime.Now().Add(-(cl.p.MaxAge - time.Second)),
		},
	}

	for _, c := range list {
		cl.Add(c)
	}

	cl.Cleanup()

	for i, c := range list {
		if i%2 == 0 {
			if _, exists := cl.list[*c.GetPartner().PartnerId()]; exists {
				t.Errorf("Connection #%d exists while being stale.", i)
			}
			if !c.closed {
				t.Errorf("Connection #%d was not closed.", i)
			}
		} else {
			if _, exists := cl.list[*c.GetPartner().PartnerId()]; !exists {
				t.Errorf("Connection #%d was removed when it was not stale.", i)
			}
			if c.closed {
				t.Errorf("Connection #%d was closed.", i)
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// Parameters                                                                 //
////////////////////////////////////////////////////////////////////////////////

// Tests that DefaultConnectionListParams returns a ConnectionListParams with
// the expected default values.
func TestDefaultConnectionListParams(t *testing.T) {
	expected := ConnectionListParams{
		CleanupPeriod: cleanupPeriodDefault,
		MaxAge:        maxAgeDefault,
	}

	p := DefaultConnectionListParams()

	if !reflect.DeepEqual(expected, p) {
		t.Errorf("Default ConnectionListParams does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, p)
	}
}
