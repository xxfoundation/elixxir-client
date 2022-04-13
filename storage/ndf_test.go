///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package storage

import (
	"reflect"
	"testing"
)

func TestSession_SetGetNDF(t *testing.T) {
	sess := InitTestingSession(t)
	testNdf := getNDF()
	sess.SetNDF(testNdf)

	if !reflect.DeepEqual(testNdf, sess.GetNDF()) {
		t.Errorf("SetNDF error: "+
			"Unexpected value after setting ndf:"+
			"Expected: %v\n\tReceived: %v", testNdf, sess.GetNDF())
	}

	receivedNdf := sess.GetNDF()
	if !reflect.DeepEqual(testNdf, receivedNdf) {
		t.Errorf("GetNDF error: "+
			"Unexpected value retrieved from GetNdf:"+
			"Expected: %v\n\tReceived: %v", testNdf, receivedNdf)

	}
}
