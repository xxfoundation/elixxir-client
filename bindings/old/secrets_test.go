////////////////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                                       //
//                                                                                        //
// Use of this source code is governed by a license that can be found in the LICENSE file //
////////////////////////////////////////////////////////////////////////////////////////////

package old

import (
	"bytes"
	"testing"
)

func TestGenerateSecret(t *testing.T) {
	secret1 := GenerateSecret(32)
	secret2 := GenerateSecret(32)

	if bytes.Compare(secret1, secret2) == 0 {
		t.Errorf("GenerateSecret: Not generating entropy")
	}

	// This runs after the test function and errors out if no panic was
	// raised.
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("GenerateSecret: Low entropy was permitted")
		}
	}()
	GenerateSecret(31)
}
