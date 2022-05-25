////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"bytes"
	"encoding/json"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Tests that no data is lost when marshaling and
// unmarshaling the Params object.
func TestParams_MarshalUnmarshal(t *testing.T) {
	// Construct a set of params
	p := GetDefaultParams()

	// Marshal the params
	data, err := json.Marshal(&p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("%s", string(data))

	// Unmarshal the params object
	received := Params{}
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Re-marshal this params object
	data2, err := json.Marshal(received)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("%s", string(data2))

	// Check that they match (it is done this way to avoid
	// false failures with the reflect.DeepEqual function and
	// pointers)
	if !bytes.Equal(data, data2) {
		t.Fatalf("Data was lost in marshal/unmarshal.")
	}
}

func TestCMIXParams_JSON_Marshal_Unmarshal(t *testing.T) {
	p := CMIXParams{
		RoundTries:  5,
		Timeout:     3 * time.Second,
		RetryDelay:  2 * time.Nanosecond,
		SendTimeout: 24 * time.Hour,
		DebugTag:    "Tag",
		BlacklistedNodes: NodeMap{
			*id.NewIdFromString("node0", id.Node, t): true,
			*id.NewIdFromString("node1", id.Node, t): true,
			*id.NewIdFromString("node2", id.Node, t): false,
			*id.NewIdFromString("node3", id.Node, t): true,
		},
		Critical: true,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Errorf("Failed to JSON marshal CMIXParams: %+v", err)
	}

	var newP CMIXParams
	err = json.Unmarshal(data, &newP)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal CMIXParams: %+v", err)
	}

	if !reflect.DeepEqual(p, newP) {
		t.Errorf("Marshalled and unmarshalled CMIXParams does not match "+
			"original.\nexpected: %+v\nreceived: %+v", p, newP)
	}
}

func TestNodeMap_MarshalJSON_UnmarshalJSON(t *testing.T) {
	nm := NodeMap{
		*id.NewIdFromString("node0", id.Node, t): true,
		*id.NewIdFromString("node1", id.Node, t): true,
		*id.NewIdFromString("node2", id.Node, t): false,
		*id.NewIdFromString("node3", id.Node, t): true,
	}

	data, err := json.Marshal(nm)
	if err != nil {
		t.Errorf("Failed to JSON marshal NodeMap: %+v", err)
	}

	newNM := make(NodeMap)
	err = json.Unmarshal(data, &newNM)
	if err != nil {
		t.Errorf("Failed to JSON unmarshal NodeMap: %+v", err)
	}

	if !reflect.DeepEqual(nm, newNM) {
		t.Errorf("Marshalled and unmarshalled NodeMap does not match original."+
			"\nexpected: %v\nreceived: %v", nm, newNM)
	}
}
