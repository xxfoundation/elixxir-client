////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package params

import "testing"

// New params path
func TestGetNetworkParameters(t *testing.T) {
	p := GetDefaultNetwork()

	expected := p.MaxCheckedRounds + 1
	p.MaxCheckedRounds = expected
	jsonString, err := p.Marshal()
	if err != nil {
		t.Errorf("%+v", err)
	}

	q, err := GetNetworkParameters(string(jsonString))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.MaxCheckedRounds != expected {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.MaxCheckedRounds, expected)
	}
}

// No new params path
func TestGetNetworkParameters_Default(t *testing.T) {
	p := GetDefaultNetwork()

	q, err := GetNetworkParameters("")
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.MaxCheckedRounds != p.MaxCheckedRounds {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.MaxCheckedRounds, p.MaxCheckedRounds)
	}
}
