////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/ekv"
)

// Ensure EkvLocalStore adheres to LocalStore.
var _ LocalStore = &EkvLocalStore{}

// Smoke test for EkvLocalStore that executes every method of LocalStore.
//
// As of writing, EkvLocalStore heavily utilizes the ekv.KeyValue
// implementation. As such, testing is light touch as heavier testing exists
// within the dependency.
func TestEkvLocalStore_Smoke(t *testing.T) {
	path := "test.txt"
	data := []byte("Test string.")

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct local store
	localStore := NewKVFilesystem(kv)

	// Write to file
	require.NoError(t, localStore.Write(path, data))

	// Read file
	read, err := localStore.Read(path)
	require.NoError(t, err)

	// Ensure read data matches originally written data
	require.Equal(t, data, read)
}

// Tests that when calling EkvLocalStore.Write, EkvLocalStore.keyLists is
// modified.
func TestEkvLocalStore_Write_KeyList(t *testing.T) {
	path := "test.txt"
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct local store
	localStore := NewKVFilesystem(kv)

	// Write data to local store
	for i := 0; i < numTests; i++ {
		curPath := path + "/" + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))

		store := localStore.(*KVFilesystem)

		// Ensure key list has been modified
		require.Contains(t, store.files, curPath)
	}

}

// Unit test for EkvLocalStore.GetList.
func TestEkvLocalStore_GetList(t *testing.T) {
	path := "test.txt"
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct local store
	localStore := NewKVFilesystem(kv)

	// Write data to local store
	expected := make(map[string]struct{}, 0)
	for i := 0; i < numTests; i++ {
		curPath := path + "/" + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))
		expected[curPath] = struct{}{}
	}

	store := localStore.(*KVFilesystem)
	received := store.files
	require.Equal(t, expected, received)
}

// Tests that loading an EkvLocalStore will load the key list.
func TestEkvLocalStore_Loading(t *testing.T) {
	path := "test.txt"
	const numTests = 100

	// Construct kv
	kv := ekv.MakeMemstore()

	// Construct local store
	localStore := NewKVFilesystem(kv)

	// Write data to local store
	for i := 0; i < numTests; i++ {
		curPath := path + "/" + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))
	}

	loadedLocalStore := NewKVFilesystem(kv)
	orig := localStore.(*KVFilesystem)
	loaded := loadedLocalStore.(*KVFilesystem)

	require.Equal(t, orig.files, loaded.files)
}
