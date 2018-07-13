////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"os"
	"reflect"
	"testing"
)

func TestInitStorage(t *testing.T) {
	TestData := []byte{12, 14, 54}
	TestSaveLoc := "testStorage.data"
	// Test with existing storage
	LocalStorage = new(RamStorage)
	// err := InitStorage(nil, "")
	// if err == nil {
	// 	t.Errorf("InitStorage failed to fail with existing storage")
	// }
	// LocalStorage = nil

	// Test DefaultStorage initialization without existing storage
	LocalStorage = nil
	err := InitStorage(nil, TestSaveLoc)
	if LocalStorage == nil {
		t.Errorf("InitStorage failed to create a default storage")
	}

	// Check that DS file location is correct
	//jww.ERROR.Printf("Default Storage file location: %v",
	//	LocalStorage.GetLocation())

	// Test DS save
	err = LocalStorage.Save(TestData)
	if err != nil {
		t.Errorf("ds.Save failed to create a save file at: %v",
			LocalStorage.GetLocation())
	}
	// Check that save file was made
	if !exists(TestSaveLoc) {
		t.Errorf("ds.Save failed to create a save file at: %v",
			TestSaveLoc)
	}
	// Test DS load
	actualData := LocalStorage.Load()
	if reflect.DeepEqual(actualData, TestData) != true {
		t.Errorf("ds.Load failed to load expected data. Expected:%v Actual:%v",
			TestData, actualData)
	}
	LocalStorage = nil

	// Test RamStorage
	LocalStorage = nil
	store := RamStorage{}
	err = InitStorage(&store, "")
	if err != nil {
		t.Errorf("InitStorage failed to initialize a RamStorage: %s", err.Error())
	}
	actualData = nil
	LocalStorage.Save(TestData)
	actualData = LocalStorage.Load()
	if reflect.DeepEqual(actualData, TestData) != true {
		t.Errorf("rs.Load failed to load expected data. Expected:%v Actual:%v",
			TestData, actualData)
	}
	os.Remove(TestSaveLoc)

	LocalStorage = nil

}

// exists returns whether the given file or directory exists or not
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}


func TestDefaultStorage_GetLocation(t *testing.T) {
	location := "hi"

	ds := DefaultStorage{location}

	if ds.GetLocation()!=location{
		t.Errorf("defaultStorage.GetLocation returned incorrect location. Expected:%v Actual:%v",
			location, ds.GetLocation())
	}
}

func TestRamStorage_GetLocation(t *testing.T) {
	location := ""

	ds := RamStorage{}

	if ds.GetLocation()!=location{
		t.Errorf("RamStorage.GetLocation returned incorrect location. Expected:%v Actual:%v",
			location, ds.GetLocation())
	}
}