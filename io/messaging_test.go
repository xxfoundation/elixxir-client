////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"testing"
)

func TestListeners(t *testing.T) {
	b := Listen(25)
	if len(listeners) == 0 {
		t.Errorf("Failed to add a listener")
	}
	StopListening(25)
	if len(listeners) != 0 {
		t.Errorf("Failed to stop listening")
	}
}
