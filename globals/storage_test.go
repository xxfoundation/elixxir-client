////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"os"
	"testing"
	//jww "github.com/spf13/jwalterweatherman"
	"reflect"
)

func TestInitStorage(t *testing.T) {
	TestData := []byte{12, 14, 54}
	TestSaveLoc := "testStorage.data"
	// Test with existing storage
	LocalStorage = new(RamStorage)
	if InitStorage(nil, "") {
		t.Errorf("InitStorage failed to fail with existing storage")
	}

	// Test DefaultStorage initialization without existing storage
	LocalStorage = nil
	InitStorage(nil, TestSaveLoc)
	if LocalStorage == nil {
		t.Errorf("InitStorage failed to create a default storage")
	}

	// Check that DS file location is correct
	//jww.ERROR.Printf("Default Storage file location: %v",
	//	LocalStorage.GetLocation())

	// Test DS save
	err := error(nil)
	LocalStorage, err = LocalStorage.Save(TestData)
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

	// Test RamStorage
	LocalStorage = nil
	store := RamStorage{}
	suc := InitStorage(store, "")
	if suc != true {
		t.Errorf("InitStorage failed to initialize a RamStorage")
	}
	actualData = nil
	LocalStorage, _ = LocalStorage.Save(TestData)
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
