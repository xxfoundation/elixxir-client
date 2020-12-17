///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"testing"
	"time"
)

func TestGetDefaultCMIX(t *testing.T) {
	c := GetDefaultCMIX()
	if c.RoundTries != 10 || c.Timeout != 25*time.Second {
		t.Errorf("GetDefaultCMIX did not return expected values")
	}
}
