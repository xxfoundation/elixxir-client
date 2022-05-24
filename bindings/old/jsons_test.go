///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"fmt"
	"testing"
)

func TestDownloadErrorDB(t *testing.T) {
	json, err := DownloadErrorDB()
	if err != nil {
		t.Errorf("DownloadErrorDB returned error: %s", err)
	}
	fmt.Printf("json: %s\n", string(json))
}

func TestDownloadDAppRegistrationDB(t *testing.T) {
	json, err := DownloadDAppRegistrationDB()
	if err != nil {
		t.Errorf("DownloadDAppRegistrationDB returned error: %s", err)
	}
	fmt.Printf("json: %s\n", string(json))
}
