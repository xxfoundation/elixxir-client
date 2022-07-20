///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Tests that no data is lost when marshaling and unmarshalling the CMIXParams
// object.
func TestParams_MarshalUnmarshal(t *testing.T) {
	// Construct a set of params
	p := GetDefaultCMixParams()

	// Marshal the params
	data, err := json.Marshal(&p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("%s", string(data))

	// Unmarshal the params object
	received := CMIXParams{}
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	// Re-marshal this params object
	data2, err := json.Marshal(received)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("%s", string(data2))

	// Check that they match (it is done this way to avoid false failures when
	// using the reflect.DeepEqual function and pointers)
	if !bytes.Equal(data, data2) {
		t.Fatalf("Data was lost in marshal/unmarshal.")
	}
}
