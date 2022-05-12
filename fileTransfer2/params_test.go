////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"gitlab.com/elixxir/client/cmix"
	"reflect"
	"testing"
)

// Tests that DefaultParams returns a Params object with the expected defaults.
func TestDefaultParams(t *testing.T) {
	expected := Params{
		MaxThroughput: defaultMaxThroughput,
		SendTimeout:   defaultSendTimeout,
		Cmix:          cmix.GetDefaultCMIXParams(),
	}
	received := DefaultParams()
	received.Cmix.Stop = expected.Cmix.Stop

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Received Params does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}
