////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestServicesManager_Add_DeleteService(t *testing.T) {
	s := NewServices()

	testId := id.NewIdFromUInt(0, id.User, t)
	testService := Service{
		Identifier: testId.Bytes(),
		Tag:        sih.Default,
	}
	s.AddService(testId, testService, nil)

	if s.numServices != 1 {
		t.Errorf("Expected successful service add increment")
	}
	if len(s.tmap[*testId]) != 1 {
		t.Errorf("Expected successful service add")
	}

	s.DeleteService(testId, testService, nil)

	if s.numServices != 0 {
		t.Errorf("Expected successful service remove decrement")
	}
	if len(s.tmap[*testId]) != 0 {
		t.Errorf("Expected successful service remove")
	}
}
