///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package conversation

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"testing"
)

// TestNewBuff tests the creation of a Buff object.
func TestNewBuff(t *testing.T) {
	kv := versioned.NewKV(make(ekv.Memstore))
	buffLen := 20
	testBuff, err := NewBuff(kv, buffLen)
	if err != nil {
		t.Fatalf("NewBuff error: %v", err)
	}

	if len(testBuff.buff) != buffLen {
		t.Fatalf("NewBuff did not produce buffer of "+
			"expected size. "+
			"\n\tExpected: %d"+
			"\n\tReceived slice size: %v",
			buffLen, len(testBuff.lookup))
	}

}
