////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
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

// Tests that ServiceList.DeepCopy returns an exact copy of a ServiceList
// without any of its elements being copies.
func TestServiceList_DeepCopy(t *testing.T) {
	sl := make(ServiceList)
	for i := 0; i < 5; i++ {
		uid := id.NewIdFromUInt(uint64(i), id.User, t)
		sl[*uid] = make([]Service, 3)
		for j := range sl[*uid] {
			sl[*uid][j] = Service{
				Identifier: []byte(fmt.Sprintf("Identifier %d %d", i, j)),
				Tag:        fmt.Sprintf("Tag %d %d", i, j),
				Metadata:   []byte(fmt.Sprintf("Metadata %d %d", i, j)),
			}
		}
	}

	newSl := sl.DeepCopy()

	if !reflect.DeepEqual(sl, newSl) {
		t.Errorf("Copy does not match original.\nexpected: %v\nreceived: %v",
			sl, newSl)
	}

	for uid := range sl {
		for j := range sl[uid] {
			if &sl[uid][j].Identifier[0] == &newSl[uid][j].Identifier[0] {
				t.Errorf("Identifier for ID %s, %d are not deep copies."+
					"\noriginal pointer: %p\ncopied pointer:   %p", &uid, j,
					sl[uid][j].Identifier, newSl[uid][j].Identifier)
			}
			if &sl[uid][j].Metadata[0] == &newSl[uid][j].Metadata[0] {
				t.Errorf("Metadata for ID %s, %d are not deep copies."+
					"\noriginal pointer: %p\ncopied pointer:   %p", &uid, j,
					sl[uid][j].Metadata, newSl[uid][j].Metadata)
			}
		}
	}
}

// Tests that CompressedServiceList.DeepCopy returns an exact copy of a
// CompressedServiceList without any of its elements being copies.
func TestCompressedServiceList_DeepCopy(t *testing.T) {
	csl := make(CompressedServiceList)
	for i := 0; i < 5; i++ {
		uid := id.NewIdFromUInt(uint64(i), id.User, t)
		csl[*uid] = make([]CompressedService, 3)
		for j := range csl[*uid] {
			csl[*uid][j] = CompressedService{
				Identifier: []byte(fmt.Sprintf("Identifier %d %d", i, j)),
				Tags: []string{
					fmt.Sprintf("Tag 1 %d %d", i, j),
					fmt.Sprintf("Tag 2 %d %d", i, j),
					fmt.Sprintf("Tag 3 %d %d", i, j),
				},
				Metadata: []byte(fmt.Sprintf("Metadata %d %d", i, j)),
			}
		}
	}

	newSl := csl.DeepCopy()

	if !reflect.DeepEqual(csl, newSl) {
		t.Errorf("Copy does not match original.\nexpected: %v\nreceived: %v",
			csl, newSl)
	}

	for uid := range csl {
		for j := range csl[uid] {
			if &csl[uid][j].Identifier[0] == &newSl[uid][j].Identifier[0] {
				t.Errorf("Identifier for ID %s, %d are not deep copies."+
					"\noriginal pointer: %p\ncopied pointer:   %p", &uid, j,
					csl[uid][j].Identifier, newSl[uid][j].Identifier)
			}
			if &csl[uid][j].Tags[0] == &newSl[uid][j].Tags[0] {
				t.Errorf("Tags for ID %s, %d are not deep copies."+
					"\noriginal pointer: %p\ncopied pointer:   %p", &uid, j,
					csl[uid][j].Tags, newSl[uid][j].Tags)
			}
			if &csl[uid][j].Metadata[0] == &newSl[uid][j].Metadata[0] {
				t.Errorf("Metadata for ID %s, %d are not deep copies."+
					"\noriginal pointer: %p\ncopied pointer:   %p", &uid, j,
					csl[uid][j].Metadata, newSl[uid][j].Metadata)
			}
		}
	}
}
