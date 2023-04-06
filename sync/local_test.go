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
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

// Smoke test for EkvLocalStore that executes Read/Write methods of LocalStore.
func TestEkvLocalStore_Smoke(t *testing.T) {
	path := "test.txt"
	data := []byte("Test string.")

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Construct local store
	localStore, err := NewOrLoadEkvLocalStore(kv)
	require.NoError(t, err)

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
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Construct local store
	localStore, err := NewOrLoadEkvLocalStore(kv)
	require.NoError(t, err)

	// Write data to local store
	for i := 0; i < numTests; i++ {
		curPath := path + LocalStoreKeyDelimiter + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))

		// check that list has been modified
		listElem, key := path, strconv.Itoa(i)

		// Ensure key list has been modified
		require.Contains(t, localStore.keyLists, listElem)
		require.Contains(t, localStore.keyLists[listElem], key)
	}

}

// Unit test for EkvLocalStore.GetList.
func TestEkvLocalStore_GetList(t *testing.T) {
	path := "test.txt"
	const numTests = 100

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Construct local store
	localStore, err := NewOrLoadEkvLocalStore(kv)
	require.NoError(t, err)

	// Write data to local store
	expected := make(KeyValueMap, 0)
	for i := 0; i < numTests; i++ {
		curPath := path + LocalStoreKeyDelimiter + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))
		expected[curPath] = data
	}

	received, err := localStore.GetList(path)
	require.NoError(t, err)
	require.Equal(t, expected, received)
}

// Tests that loading an EkvLocalStore will load the key list.
func TestEkvLocalStore_Loading(t *testing.T) {
	path := "test.txt"
	const numTests = 100

	// Construct kv
	kv := versioned.NewKV(ekv.MakeMemstore())

	// Construct local store
	localStore, err := NewOrLoadEkvLocalStore(kv)
	require.NoError(t, err)

	// Write data to local store
	for i := 0; i < numTests; i++ {
		curPath := path + LocalStoreKeyDelimiter + strconv.Itoa(i)
		data := []byte("Test string." + strconv.Itoa(i))
		require.NoError(t, localStore.Write(curPath, data))
	}

	loadedLocalStore, err := NewOrLoadEkvLocalStore(kv)
	require.NoError(t, err)
	require.Equal(t, localStore.keyLists, loadedLocalStore.keyLists)
}
