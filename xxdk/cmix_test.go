////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	remoteSync "gitlab.com/elixxir/client/v4/sync"
	"gitlab.com/xx_network/primitives/netTime"
)

func TestSynchronized(t *testing.T) {
	def := getNDF(t)
	marshalledDef, _ := def.Marshal()
	storageDir := ".TestSynchronized"
	sinfo, err := os.Stat(storageDir)
	if os.IsExist(err) || (sinfo != nil && sinfo.IsDir()) {
		os.RemoveAll(storageDir)
	}
	password := []byte("hunter2")

	err = NewCmix(string(marshalledDef), storageDir, password, "AAAA")
	require.NoError(t, err)

	// We can't call Load, because that trys to make a real net connection
	_, err = OpenCmix(storageDir, password)
	require.NoError(t, err)

	//Now upgrade using Synchronized
	syncPrefixes := []string{"sync", "a", "abcdefghijklmnop", "b", "c"}
	remoteCallCnt := 0
	updateCb := func(newTx remoteSync.Mutate, err error) {
		t.Logf("KEY: %s, VAL: %s", newTx.Key, newTx.Value)
		remoteCallCnt += 1
	}
	remote := &mockRemote{
		data: make(map[string][]byte, 0),
	}
	_, err = OpenSynchronizedCmix(storageDir, password, remote,
		syncPrefixes, nil, updateCb)
	require.NoError(t, err)

	// Initialize once more to show that it can init more than once
	_, err = OpenSynchronizedCmix(storageDir, password, remote,
		syncPrefixes, nil, updateCb)
	require.NoError(t, err)

	// Now ensure re-opening the old way causes a panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
		os.RemoveAll(storageDir)
	}()
	OpenCmix(storageDir, password)
	t.Error("OpenCmix worked without panic")
}

type mockRemote struct {
	lck  sync.Mutex
	data map[string][]byte
}

func (m *mockRemote) Read(path string) ([]byte, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	return m.data[path], nil
}

func (m *mockRemote) Write(path string, data []byte) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.data[path] = append(m.data[path], data...)
	return nil
}

func (m *mockRemote) ReadDir(path string) ([]string, error) {
	panic("unimplemented")
}

func (m mockRemote) GetLastModified(path string) (time.Time, error) {
	return netTime.Now(), nil
}

func (m mockRemote) GetLastWrite() (time.Time, error) {
	return netTime.Now(), nil
}
