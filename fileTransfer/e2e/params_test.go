////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Tests that DefaultParams returns a Params object with the expected defaults.
func TestDefaultParams(t *testing.T) {
	expected := Params{
		NotifyUponCompletion: defaultNotifyUponCompletion,
	}

	received := DefaultParams()
	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Received Params does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Tests that GetParameters uses the passed in parameters.
func TestGetParameters(t *testing.T) {
	expected := Params{
		NotifyUponCompletion: false,
	}
	expectedData, err := json.Marshal(expected)
	if err != nil {
		t.Errorf("Failed to JSON marshal expected params: %+v", err)
	}

	p, err := GetParameters(string(expectedData))
	if err != nil {
		t.Errorf("Failed get parameters: %+v", err)
	}

	if !reflect.DeepEqual(expected, p) {
		t.Errorf("Received Params does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, p)
	}
}

// Tests that GetParameters returns the default parameters if no params string
// is provided
func TestGetParameters_Default(t *testing.T) {
	expected := DefaultParams()

	p, err := GetParameters("")
	if err != nil {
		t.Errorf("Failed get parameters: %+v", err)
	}

	if !reflect.DeepEqual(expected, p) {
		t.Errorf("Received Params does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, p)
	}
}

// Error path: Tests that GetParameters returns an error when the params string
// does not contain a valid JSON representation of Params.
func TestGetParameters_InvalidParamsStringError(t *testing.T) {
	_, err := GetParameters("invalid JSON")
	if err == nil {
		t.Error("Failed get get error for invalid JSON")
	}
}

// Tests that a Params object marshalled via json.Marshal and unmarshalled via
// json.Unmarshal matches the original.
func TestParams_JsonMarshalUnmarshal(t *testing.T) {
	// Construct a set of params
	expected := DefaultParams()

	// Marshal the params
	data, err := json.Marshal(&expected)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	// Unmarshal the params object
	received := Params{}
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Marshalled and unmarshalled Params does not match original."+
			"\nexpected: %#v\nreceived: %#v", expected, received)
	}
}
