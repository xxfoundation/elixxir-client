///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"testing"
	"time"
)

func TestGetDefaultCMIX(t *testing.T) {
	c := GetDefaultCMIX()
	if c.RoundTries != 10 || c.Timeout != 25*time.Second {
		t.Errorf("GetDefaultCMIX did not return expected values")
	}
}

// New params path
func TestGetCMIXParameters(t *testing.T) {
	p := GetDefaultCMIX()

	expected := p.RoundTries + 1
	p.RoundTries = expected
	jsonString, err := p.Marshal()
	if err != nil {
		t.Errorf("%+v", err)
	}

	q, err := GetCMIXParameters(string(jsonString))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != expected {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, expected)
	}
}

// No new params path
func TestGetCMIXParameters_Default(t *testing.T) {
	p := GetDefaultCMIX()

	q, err := GetCMIXParameters("")
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != p.RoundTries {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, p.RoundTries)
	}
}
