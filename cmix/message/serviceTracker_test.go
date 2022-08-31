////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestServiceList_Marshal_UnmarshalJSON(t *testing.T) {
	var sl ServiceList = make(map[id.ID][]Service)
	numServices := 3
	testString := "test"
	for i := 0; i < numServices; i++ {
		uid := id.NewIdFromUInt(uint64(i), id.User, t)
		sl[*uid] = []Service{{Tag: testString}}
	}
	jsonResult, err := sl.MarshalJSON()
	if err != nil {
		t.Errorf(err.Error())
	}

	t.Logf("%s", jsonResult)

	sl = make(map[id.ID][]Service)
	err = sl.UnmarshalJSON(jsonResult)
	if err != nil {
		t.Errorf(err.Error())
	}

	if len(sl) != numServices {
		t.Errorf("Unexpected # of services: Got %d, expected %d", len(sl), numServices)
	}
	for _, newService := range sl {
		if newService[0].Tag != testString {
			t.Errorf("Unexpected service tag: Got %s, expected %s", newService[0].Tag, testString)
		}
	}
}
