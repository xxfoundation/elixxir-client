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

	// Test DefaultStorage initialization without existing storage
	storage := &DefaultStorage{}
	storage.SetLocation(TestSaveLoc)

	// Check that DS file location is correct
	//jww.ERROR.Printf("Default Storage file location: %v",
	//	LocalStorage.GetLocation())

	// Test DS save
	err := storage.Save(TestData)
	if err != nil {
		t.Errorf("ds.Save failed to create a save file at: %v",
			storage.GetLocation())
	}
	// Check that save file was made
	if !exists(TestSaveLoc) {
		t.Errorf("ds.Save failed to create a save file at: %v",
			TestSaveLoc)
	}
	// Test DS load
	actualData := storage.Load()
	if reflect.DeepEqual(actualData, TestData) != true {
		t.Errorf("ds.Load failed to load expected data. Expected:%v Actual:%v",
			TestData, actualData)
	}

	// Test RamStorage
	store := RamStorage{}
	actualData = nil
	store.Save(TestData)
	actualData = store.Load()
	if reflect.DeepEqual(actualData, TestData) != true {
		t.Errorf("rs.Load failed to load expected data. Expected:%v Actual:%v",
			TestData, actualData)
	}
	os.Remove(TestSaveLoc)
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

	ds := DefaultStorage{location: location}

	if ds.GetLocation() != location {
		t.Errorf("defaultStorage.GetLocation returned incorrect location. Expected:%v Actual:%v",
			location, ds.GetLocation())
	}
}

func TestRamStorage_GetLocation(t *testing.T) {
	location := ""

	ds := RamStorage{}

	if ds.GetLocation() != location {
		t.Errorf("RamStorage.GetLocation returned incorrect location. Expected:%v Actual:%v",
			location, ds.GetLocation())
	}
}
