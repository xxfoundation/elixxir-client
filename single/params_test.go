////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/cmix"
	"reflect"
	"testing"
)

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
