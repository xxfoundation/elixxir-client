package e2e

// Testing file for the params.go functions

import "testing"

// Test that the GetDefaultParams function returns the right default data
func Test_GetDefaultParams(t *testing.T) {
	p := GetDefaultSessionParams()
	if p.MinKeys != minKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.MinKeys, minKeys)
	}
	if p.MaxKeys != maxKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.MaxKeys, maxKeys)
	}
	if p.NumRekeys != numReKeys {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.NumRekeys, numReKeys)
	}
	if p.TTLScalar != ttlScalar {
		t.Errorf("MinKeys mismatch\r\tGot: %v\r\tExpected: %v", p.TTLScalar, ttlScalar)
	}
	if p.MinNumKeys != threshold {
		t.Errorf("MinKeys mismatch\r\tGot: %d\r\tExpected: %d", p.MinNumKeys, threshold)
	}
}
