////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package backup

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"testing"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

func Test_storeJson_loadJson(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	json := "{'data': {'one': 1}}"

	err := storeJson(json, kv)
	if err != nil {
		t.Errorf("Failed to store JSON: %+v", err)
	}

	loaded := loadJson(kv)
	if loaded != json {
		t.Errorf("Did not receive expected data from KV.\n\tExpected: %s, Received: %s\n", json, loaded)
	}
}
