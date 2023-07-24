////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"bytes"
	"encoding/json"
	"testing"

	"gitlab.com/elixxir/client/v4/xxdk"
)

// Tests that xxdk.E2EParams can be JSON marshalled and unmarshalled.
func TestE2EParams_JsonMarshalUnmarshal(t *testing.T) {
	// Construct a set of params
	expected := xxdk.GetDefaultE2EParams()

	// Marshal the params
	data, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("Failed to JSON marshal E2EParams: %+v", err)
	}

	// Unmarshal the params object
	var received xxdk.E2EParams
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Failed to JSON unmarshal E2EParams: %+v", err)
	}

	// Re-marshal this params object
	data2, err := json.Marshal(received)
	if err != nil {
		t.Fatalf("Failed to re-JSON marshal E2EParams: %+v", err)
	}

	// Check that the JSON matches (this is done to avoid false failures with
	// the reflect.DeepEqual function and pointers)
	if !bytes.Equal(data, data2) {
		t.Fatalf("Data was lost in JSON marshal/unmarshal."+
			"\nexpected: %s\nreceived: %s", data, data2)
	}
}
