////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package params

import "testing"

// New params path
func TestGetUnsafeParameters(t *testing.T) {
	p := GetDefaultUnsafe()

	expected := p.RoundTries + 1
	p.RoundTries = expected
	jsonString, err := p.Marshal()
	if err != nil {
		t.Errorf("%+v", err)
	}

	q, err := GetUnsafeParameters(string(jsonString))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != expected {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, expected)
	}
}

// No new params path
func TestGetUnsafeParameters_Default(t *testing.T) {
	p := GetDefaultUnsafe()

	q, err := GetUnsafeParameters("")
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != p.RoundTries {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, p.RoundTries)
	}
}
