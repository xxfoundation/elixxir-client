////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package bindings

import (
	"testing"
)

// Mock dummy storage interface for testing.
type DummyStorage struct {
	Location string
	LastSave []byte
}

func (d DummyStorage) SetLocation(l string) (*Storage, error) {
	d.Location = l
	return &d, nil
}

func (d DummyStorage) GetLocation() string {
	return d.Location
}

func (d DummyStorage) Save(b []byte) (*Storage, error) {
	d.LastSave = make([]byte, len(b))
	for i = 0; i < len(b); i++ {
		d.LastSave[i] = b[i]
	}
	return &d, nil
}

func (d DummyStorage) Load() []byte {
	return d.LastSave
}


// Make sure InitClient returns an error when called incorrectly.
func TestInitClientNil(t *testing.T) {
	err := InitClient(nil, nil)
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, nil) input!")
	}
	err := InitClient(nil, "hello")
	if err == nil {
		t.Errorf("InitClient returned nil on invalid (nil, 'hello') input!")
	}
}

func TestInitClient(t *testing.T) {
	d := DummyStorage{Location: "Blah", LastSave: []byte{'a', 'b', 'c'}}
	err := InitClient(&d, "hello")
	if err != nil {
		t.Errorf("InitClient returned error: %v", err)
	"strings"
)

func TestGetContactListJSON(t *testing.T) {
	// This call includes validating the JSON against the schema
	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err.Error())
	}

	// But, just in case,
	// let's make sure that we got the error out of validateContactList anyway
	err = validateContactListJSON(result)

	if err != nil {
		t.Error(err.Error())
	}

	// Finally, make sure that all the names we expect are in the JSON
	expected := []string{"Ben", "Rick", "Jake", "Mario", "Allan", "David",
	"Jim", "Spencer", "Will", "Jono"}

	actual := string(result)

	for _, nick := range(expected) {
		if !strings.Contains(actual, nick) {
			t.Errorf("Error: Expected name %v wasn't in JSON %v", nick, actual)
		}
	}
}
