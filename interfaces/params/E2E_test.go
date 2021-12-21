///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import "testing"

func TestGetDefaultE2E(t *testing.T) {
	if GetDefaultE2E().Type != Standard {
		t.Errorf("GetDefaultE2E did not return Standard")
	}
	if !GetDefaultE2E().OnlyNotifyOnLastSend  {
		t.Errorf("GetDefaultE2E did not return OnlyNotifyOnLastSend == true")
	}
}

func TestSendType_String(t *testing.T) {
	e := E2E{Type: Standard}
	if e.Type.String() != "Standard" {
		t.Errorf("Running String on Standard E2E type got %s", e.Type.String())
	}

	e = E2E{Type: KeyExchange}
	if e.Type.String() != "KeyExchange" {
		t.Errorf("Running String on KeyExchange E2E type got %s", e.Type.String())
	}

	e = E2E{Type: SendType(40)}
	if e.Type.String() != "Unknown SendType 40" {
		t.Errorf("Running String on unknown E2E type got %s", e.Type.String())
	}
}

// New params path
func TestGetE2EParameters(t *testing.T) {
	p := GetDefaultE2E()

	expected := p.RoundTries + 1
	p.RoundTries = expected
	jsonString, err := p.Marshal()
	if err != nil {
		t.Errorf("%+v", err)
	}

	q, err := GetE2EParameters(string(jsonString))
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != expected {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, expected)
	}
}

// No new params path
func TestGetE2EParameters_Default(t *testing.T) {
	p := GetDefaultE2E()

	q, err := GetE2EParameters("")
	if err != nil {
		t.Errorf("%+v", err)
	}

	if q.RoundTries != p.RoundTries {
		t.Errorf("Parameters failed to change! Got %d, Expected %d", q.RoundTries, p.RoundTries)
	}
}

// Test that the GetDefaultParams function returns the right default data
func Test_GetDefaultParams(t *testing.T) {
	p := GetDefaultE2ESessionParams()
	if p.MinKeys != minKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.MinKeys, minKeys)
	}
	if p.MaxKeys != maxKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.MaxKeys, maxKeys)
	}
	if p.NumRekeys != numReKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.NumRekeys, numReKeys)
	}
}
