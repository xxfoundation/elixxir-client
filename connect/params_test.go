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

func TestParams_MarshalUnmarshal(t *testing.T) {
	// Construct a set of params
	p := xxdk.GetDefaultE2EParams()

	// Marshal the params
	data, err := json.Marshal(&p)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	t.Logf("%s", string(data))

	// Unmarshal the params object
	received := xxdk.E2EParams{}
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

	// Check that they match (it is done this way to avoid
	// false failures with the reflect.DeepEqual function and
	// pointers)
	if !bytes.Equal(data, data2) {
		t.Fatalf("Data was lost in marshal/unmarshal.")
	}
}
