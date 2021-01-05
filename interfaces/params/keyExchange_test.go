////////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package params

import "testing"

// New params path
func TestGetRekeyParameters(t *testing.T) {
	p := GetDefaultRekey()

	expected := p.RoundTimeout + 1
	p.RoundTimeout = expected
	jsonString, err := p.Marshal()
	if err != nil {
		t.Errorf("%+v", err)
	}

	q, err := GetRekeyParameters(string(jsonString))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTimeout != expected {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTimeout, expected)
	}
}

// No new params path
func TestGetRekeyParameters_Default(t *testing.T) {
	p := GetDefaultRekey()

	q, err := GetRekeyParameters("")
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTimeout != p.RoundTimeout {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTimeout, p.RoundTimeout)
	}
}
