////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package hostList

import (
	"fmt"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strings"
	"testing"
)

// Unit test of NewStore.
func TestNewStore(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	expected := &Store{kv: kv.Prefix(hostListPrefix)}

	s := NewStore(kv)

	if !reflect.DeepEqual(expected, s) {
		t.Errorf("NewStore did not return the expected object."+
			"\nexpected: %+v\nreceived: %+v", expected, s)
	}
}

// Tests that a host list saved by Store.Store matches the host list returned
// by Store.Get.
func TestStore_Store_Get(t *testing.T) {
	s := NewStore(versioned.NewKV(make(ekv.Memstore)))
	list := []*id.ID{
		id.NewIdFromString("histID_1", id.Node, t),
		nil,
		id.NewIdFromString("histID_2", id.Node, t),
		id.NewIdFromString("histID_3", id.Node, t),
	}

	err := s.Store(list)
	if err != nil {
		t.Errorf("Store returned an error: %+v", err)
	}

	newList, err := s.Get()
	if err != nil {
		t.Errorf("get returned an error: %+v", err)
	}

	if !reflect.DeepEqual(list, newList) {
		t.Errorf("Failed to save and load host list."+
			"\nexpected: %+v\nreceived: %+v", list, newList)
	}
}

// Error path: tests that Store.Get returns an error if not host list is
// saved in storage.
func TestStore_Get_StorageError(t *testing.T) {
	s := NewStore(versioned.NewKV(make(ekv.Memstore)))
	expectedErr := strings.SplitN(getStorageErr, "%", 2)[0]

	_, err := s.Get()
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("get failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that a list of IDs that is marshalled using marshalHostList and
// unmarshalled using unmarshalHostList matches the original.
func Test_marshalHostList_unmarshalHostList(t *testing.T) {
	list := []*id.ID{
		id.NewIdFromString("histID_1", id.Node, t),
		nil,
		id.NewIdFromString("histID_2", id.Node, t),
		id.NewIdFromString("histID_3", id.Node, t),
	}

	data := marshalHostList(list)

	newList, err := unmarshalHostList(data)
	if err != nil {
		t.Errorf("unmarshalHostList produced an error: %+v", err)
	}

	if !reflect.DeepEqual(list, newList) {
		t.Errorf("Failed to marshal and unmarshal ID list."+
			"\nexpected: %+v\nreceived: %+v", list, newList)
	}
}

// Error path: tests that unmarshalHostList returns an error if the data is not
// of the correct length.
func Test_unmarshalHostList_InvalidDataErr(t *testing.T) {
	data := []byte("Invalid Data")
	expectedErr := fmt.Sprintf(unmarshallLenErr, len(data))

	_, err := unmarshalHostList(data)
	if err == nil || err.Error() != expectedErr {
		t.Errorf("unmarshalHostList failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
