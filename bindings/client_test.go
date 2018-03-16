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
	}
}
