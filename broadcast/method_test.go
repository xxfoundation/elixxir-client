////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import "testing"

// Consistency test of Method.String.
func TestMethod_String(t *testing.T) {
	tests := map[Method]string{
		Symmetric:    "Symmetric",
		RSAToPublic:  "RSAToPublic",
		RSAToPrivate: "RSAToPrivate",
		100:          "INVALID METHOD 100",
	}

	for method, expected := range tests {
		if method.String() != expected {
			t.Errorf("Invalid string for method %d."+
				"\nexpected: %s\nreceived: %s", method, expected, method)
		}
	}
}
