///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/json"
	"testing"
)

func TestParams_MarshalUnmarshal(t *testing.T) {
	p := GetDefaultParams()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %+v", err)
	}

	t.Logf("%s", string(data))

	received := GetDefaultParams()
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Unmarshal error: %+v", err)
	}

	data2, err := json.Marshal(received)
	if err != nil {
		t.Fatalf("Marshal error: %+v", err)
	}

	t.Logf("%s", string(data2))

}
