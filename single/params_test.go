////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"encoding/json"
	"gitlab.com/elixxir/client/cmix"
	"reflect"
	"testing"
)

// Tests that no data is lost when marshaling and
// unmarshaling the RequestParams object.
func TestParams_MarshalUnmarshal(t *testing.T) {
	p := GetDefaultRequestParams()
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal error: %+v", err)
	}

	t.Logf("%s", string(data))

	received := RequestParams{}
	err = json.Unmarshal(data, &received)
	if err != nil {
		t.Fatalf("Unmarshal error: %+v", err)
	}

	data2, err := json.Marshal(received)
	if err != nil {
		t.Fatalf("Marshal error: %+v", err)
	}

	t.Logf("%s", string(data2))

	if !bytes.Equal(data, data2) {
		t.Fatalf("Data was lost in marshal/unmarshal.")
	}
}

// Tests that GetDefaultRequestParams returns a RequestParams with the expected
// default values.
func TestGetDefaultRequestParams(t *testing.T) {
	expected := RequestParams{
		Timeout:             defaultRequestTimeout,
		MaxResponseMessages: defaultMaxResponseMessages,
		CmixParams:          cmix.GetDefaultCMIXParams(),
	}

	params := GetDefaultRequestParams()
	params.CmixParams.Stop = expected.CmixParams.Stop

	if !reflect.DeepEqual(expected, params) {
		t.Errorf("Failed to get expected default params."+
			"\nexpected: %+v\nreceived: %+v", expected, params)
	}
}
