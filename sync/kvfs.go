////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"fmt"
	"sync"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/ekv"
)

const (
	// ekvFilesystemPrefix is prefixed to all file operations
	ekvFilesystemPrefix = "ekvfs"
	// The key for the tracked files
	ekvFilesystemFiles  = "ekvfs_files"
	ekvFilsystemPathFmt = "%s/%s"
)

// KVFilesystem implements filesystem ([FileIO]) operations inside the provided
// EKV instance.
type KVFilesystem struct {
	prefix string
	store  ekv.KeyValue
	files  map[string]struct{}
	lck    sync.Mutex
}

// NewKVFilesystem initializes a KVFilesystem object and returns it.
// If the KVFilesystem exists inside the ekv, it loads the file listing
// from it. If an error occurs loading the files on start it can panic.
func NewKVFilesystem(kv ekv.KeyValue) FileIO {
	return NewKVFilesystemWithPrefix(ekvFilesystemPrefix, kv)
}

// NewKVFilesystemWithPrefix creats a KVFilesystem using a custom prefix.
func NewKVFilesystemWithPrefix(prefix string, kv ekv.KeyValue) FileIO {
	fs := &KVFilesystem{
		prefix: prefix,
		store:  kv,
		files:  make(map[string]struct{}),
	}
	err := fs.loadFiles()
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return fs
}

// Read reads data from path. This will return an error if it fails to read from
// the file path.
//
// This utilizes [ekv.KeyValue] under the hood.
func (k *KVFilesystem) Read(path string) ([]byte, error) {
	key := k.makeKey(path)
	return k.store.GetBytes(key)
}

// Write writes data to the path. This will return an error if it fails to
// write.
//
// This utilizes [ekv.KeyValue] under the hood.
func (k *KVFilesystem) Write(path string, data []byte) error {

	k.lck.Lock()
	k.files[path] = struct{}{}
	k.lck.Unlock()
	err := k.saveFiles()
	if err != nil {
		return err
	}
	key := k.makeKey(path)
	return k.store.SetBytes(key, data)
}

func (k *KVFilesystem) makeKey(path string) string {
	return fmt.Sprintf(ekvFilsystemPathFmt, ekvFilesystemPrefix, path)
}

func (k *KVFilesystem) loadFiles() error {
	key := k.makeKey(ekvFilesystemFiles)
	data, err := k.store.GetBytes(key)
	if err != nil {
		// If the key doesn't exist then this is not an error
		if !ekv.Exists(err) {
			return nil
		}
		return err
	}

	k.lck.Lock()
	defer k.lck.Unlock()
	err = json.Unmarshal(data, &k.files)
	return err
}

func (k *KVFilesystem) saveFiles() error {
	key := k.makeKey(ekvFilesystemFiles)
	k.lck.Lock()
	data, err := json.Marshal(k.files)
	k.lck.Unlock()

	if err != nil {
		return err
	}

	return k.store.SetBytes(key, data)
}
