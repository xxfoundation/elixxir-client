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

package gateway

import (
	"fmt"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strings"
	"testing"
)

// Tests that a host list saved by Store.Store matches the host list returned
// by Store.Get.
func TestStore_Store_Get(t *testing.T) {
	// Init list to store
	list := []*id.ID{
		id.NewIdFromString("histID_1", id.Node, t),
		nil,
		id.NewIdFromString("histID_2", id.Node, t),
		id.NewIdFromString("histID_3", id.Node, t),
	}

	// Init storage
	testStorage := storage.InitTestingSession(t)
	storeKv := testStorage.GetKV().Prefix(hostListPrefix)

	// Save into storage
	err := saveHostList(storeKv, list)
	if err != nil {
		t.Errorf("Store returned an error: %+v", err)
	}

	// Retrieve stored data from storage
	newList, err := getHostList(storeKv)
	if err != nil {
		t.Errorf("get returned an error: %+v", err)
	}

	// Ensure retrieved data from storage matches
	// what was stored
	if !reflect.DeepEqual(list, newList) {
		t.Errorf("Failed to save and load host list."+
			"\nexpected: %+v\nreceived: %+v", list, newList)
	}
}

// Error path: tests that Store.Get returns an error if no host list is
// saved in storage.
func TestStore_Get_StorageError(t *testing.T) {

	// Init storage
	testStorage := storage.InitTestingSession(t)
	storeKv := testStorage.GetKV().Prefix(hostListPrefix)

	// Construct expected error
	expectedErr := strings.SplitN(getStorageErr, "%", 2)[0]

	// Attempt to pull from an empty store
	_, err := getHostList(storeKv)

	// Check that the expected error is received
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("get failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that a list of IDs that is marshalled using marshalHostList and
// unmarshalled using unmarshalHostList matches the original.
func Test_marshalHostList_unmarshalHostList(t *testing.T) {
	// Construct example list
	list := []*id.ID{
		id.NewIdFromString("histID_1", id.Node, t),
		nil,
		id.NewIdFromString("histID_2", id.Node, t),
		id.NewIdFromString("histID_3", id.Node, t),
	}

	// Marshal list
	data := marshalHostList(list)

	// Unmarshal marsalled data into new object
	newList, err := unmarshalHostList(data)
	if err != nil {
		t.Errorf("unmarshalHostList produced an error: %+v", err)
	}

	// Ensure original data and unmarshalled data is consistent
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
