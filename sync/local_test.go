////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

// Ensure EkvLocalStore adheres to LocalStore.
var _ LocalStore = &EkvLocalStore{}

// Smoke test for EkvLocalStore that executes every method of LocalStore.
//
// As of writing, EkvLocalStore heavily utilizes the ekv.KeyValue
// implementation. As such, testing is light touch as heavier testing exists
// within the dependency.
func TestEkvLocalStore_Smoke(t *testing.T) {
	baseDir := "testDir"
	path := "test.txt"
	data := []byte("Test string.")

	localStore, err := NewEkvLocalStore(baseDir, "password")
	require.NoError(t, err)

	// Delete the test file at the end
	defer func() {
		require.NoError(t, os.RemoveAll(baseDir))

	}()

	// Write to file
	require.NoError(t, localStore.Write(path, data))

	// Read file
	read, err := localStore.Read(path)
	require.NoError(t, err)

	// Ensure read data matches originally written data
	require.Equal(t, data, read)

}
