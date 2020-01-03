////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
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
	TestDataA := []byte{12, 14, 54}
	TestDataB := []byte{69, 42, 32}
	TestSaveLocA := "testStorageA.data"
	TestSaveLocB := "testStorageB.data"

	// Test DefaultStorage initialization without existing storage
	storage := &DefaultStorage{}
	//Check that storage is empty prior to any Save calls
	if !storage.IsEmpty() {
		t.Errorf("ds.IsEmpty failed to detect an empty storage")
	}

	storage.SetLocation(TestSaveLocA, TestSaveLocB)

	// Test DS saveA
	err := storage.SaveA(TestDataA)
	if err != nil {
		t.Errorf("ds.Save failed to create a save file A at: %v",
			TestSaveLocA)
	}
	// Check that save file was made
	if !exists(TestSaveLocA) {
		t.Errorf("ds.Save failed to create a save file A at: %v",
			TestSaveLocA)
	}
	//Check that the storage is not empty after a saveA call
	if storage.IsEmpty() {
		t.Errorf("ds.IsEmpty failed to detect a non-empty storage")
	}

	// Test DS loadA
	actualData := storage.LoadA()
	if reflect.DeepEqual(actualData, TestDataA) != true {
		t.Errorf("ds.Load failed to load expected data on A. Expected:%v Actual:%v",
			TestDataA, actualData)
	}

	// Test DS saveB
	err = storage.SaveB(TestDataB)
	if err != nil {
		t.Errorf("ds.Save failed to create a save file B at: %v",
			TestSaveLocB)
	}
	// Check that save file was made
	if !exists(TestSaveLocB) {
		t.Errorf("ds.Save failed to create a save file B at: %v",
			TestSaveLocB)
	}

	// Test DS loadA
	actualData = storage.LoadB()
	if reflect.DeepEqual(actualData, TestDataB) != true {
		t.Errorf("ds.Load failed to load expected data on B. Expected:%v Actual:%v",
			TestDataB, actualData)
	}

	// Test RamStorage
	store := RamStorage{}
	actualData = nil
	// Test A
	store.SaveA(TestDataA)
	actualData = store.LoadA()
	if reflect.DeepEqual(actualData, TestDataA) != true {
		t.Errorf("rs.Load failed to load expected data A. Expected:%v Actual:%v",
			TestDataA, actualData)
	}
	//Test B
	store.SaveB(TestDataB)
	actualData = store.LoadB()
	if reflect.DeepEqual(actualData, TestDataB) != true {
		t.Errorf("rs.Load failed to load expected data B. Expected:%v Actual:%v",
			TestDataB, actualData)
	}
	os.Remove(TestSaveLocA)
	os.Remove(TestSaveLocB)
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
	locationA := "hi"
	locationB := "hi2"

	ds := DefaultStorage{locationA: locationA, locationB: locationB}

	recievedLocA, recievedLocB := ds.GetLocation()

	if recievedLocA != locationA {
		t.Errorf("defaultStorage.GetLocation returned incorrect location A. Expected:%v Actual:%v",
			locationA, recievedLocA)
	}

	if recievedLocB != locationB {
		t.Errorf("defaultStorage.GetLocation returned incorrect location B. Expected:%v Actual:%v",
			locationB, recievedLocB)
	}
}

func TestRamStorage_GetLocation(t *testing.T) {

	ds := RamStorage{}

	a, b := ds.GetLocation()

	if a != "" && b != "" {
		t.Errorf("RamStorage.GetLocation returned incorrect location. Actual: '', ''; Expected:'%v','%v'",
			a, b)
	}
}
