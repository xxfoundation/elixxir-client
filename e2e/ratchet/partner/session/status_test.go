///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package session

// Testing file for the status.go functions

import "testing"

// Test that Status_String returns the right strings for a status
func Test_Status_String(t *testing.T) {
	if Status(Active).String() != "Active" {
		t.Errorf("Testing Active returned mismatch.\r\tGot: %s\r\tExpected: %s", Status(Active).String(), "Active")
	}
	if Status(RekeyNeeded).String() != "Rekey Needed" {
		t.Errorf("Testing RekeyNeeded returned mismatch.\r\tGot: %s\r\tExpected: %s", Status(RekeyNeeded).String(), "Rekey Needed")
	}
	if Status(Empty).String() != "Empty" {
		t.Errorf("Testing Empty returned mismatch.\r\tGot: %s\r\tExpected: %s", Status(Empty).String(), "Empty")
	}
	if Status(RekeyEmpty).String() != "Rekey Empty" {
		t.Errorf("Testing RekeyEmpty returned mismatch.\r\tGot: %s\r\tExpected: %s", Status(RekeyEmpty).String(), "Rekey Empty")
	}
}
