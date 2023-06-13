////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
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
	if len(s.services[*testId]) != 1 {
		t.Errorf("Expected successful service add")
	}

	s.DeleteService(testId, testService, nil)

	if s.numServices != 0 {
		t.Errorf("Expected successful service remove decrement")
	}
	if len(s.services[*testId]) != 0 {
		t.Errorf("Expected successful service remove")
	}
}

func TestServicesManager_Add_Delete_CompressedService(t *testing.T) {
	s := NewServices()

	testId := id.NewIdFromUInt(0, id.User, t)
	testService := CompressedService{
		Identifier: testId.Bytes(),
		Tags:       []string{sih.Default},
		Metadata:   []byte("testmetadata"),
	}
	s.UpsertCompressedService(testId, testService, nil)

	if s.numServices != 1 {
		t.Errorf("Expected successful service add increment")
	}
	if len(s.compressedServices[*testId]) != 1 {
		t.Errorf("Expected successful service add")
	}

	testService.Tags = append(testService.Tags, sih.Group)

	s.UpsertCompressedService(testId, testService, nil)

	if s.numServices != 1 {
		t.Errorf("Expected successful service add increment\n\tExpected: %d\n\tReceived: %d\n", 1, s.numServices)
	}
	if len(s.compressedServices[*testId]) != 1 {
		t.Errorf("Expected successful service add")
	}

	s.DeleteCompressedService(testId, testService, nil)

	if s.numServices != 0 {
		t.Errorf("Expected successful service remove decrement")
	}
	if len(s.compressedServices[*testId]) != 0 {
		t.Errorf("Expected successful service remove")
	}
}
