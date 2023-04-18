////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"path/filepath"

	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/sync"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

// LocalKV creates a filesystem based KV that doesn't
// synchronize with a remote storage system.
func LocalKV(storageDir string, password []byte,
	rng *fastRNG.StreamGenerator) (*sync.VersionedKV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewFilestore(storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}
	localFS := sync.NewFileSystemRemoteStorage(filepath.Join(storageDir,
		localTxLogPath))
	return sync.LocalKV(storageDir, password, localFS, localKV, rng)
}

// SynchronizedKV creates a filesystem based KV that synchronizes
// with a remote storage system.
func SynchronizedKV(storageDir string, password []byte,
	remote sync.RemoteStore,
	synchedPrefixes []string,
	eventCb sync.KeyUpdateCallback,
	updateCb sync.RemoteStoreCallback,
	rng *fastRNG.StreamGenerator) (*sync.VersionedKV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewFilestore(storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}
	localFS := sync.NewFileSystemRemoteStorage(filepath.Join(storageDir,
		localTxLogPath))

	return sync.SynchronizedKV(storageDir, password, localFS,
		remote, localKV, synchedPrefixes, eventCb, updateCb, rng)
}
