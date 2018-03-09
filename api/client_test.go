////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"testing"
)

func TestGetContactListJSON(t *testing.T) {
	// todo: flesh this test out, print actual vs expected, and so on
	result, err := GetContactListJSON()

	if err != nil {
		t.Error(err)
	}

	t.Log(string(result))
}
