////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

// LocalKV creates a filesystem based KV that doesn't
// synchronize with a remote storage system.
func LocalKV(storageDir string, password []byte,
	rng *fastRNG.StreamGenerator) (versioned.KV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewFilestore(storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}
	// localFS := collective.NewFileSystemRemoteStorage(filepath.Join(storageDir,
	// 	localTxLogPath))
	return collective.LocalKV(storageDir, string(password), localKV, rng)
}

// SynchronizedKV creates a filesystem based KV that synchronizes
// with a remote storage system.
func SynchronizedKV(storageDir string, password []byte,
	remote collective.RemoteStore,
	synchedPrefixes []string,
	eventCb collective.KeyUpdateCallback,
	updateCb collective.RemoteStoreCallback,
	rng *fastRNG.StreamGenerator) (versioned.KV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewFilestore(storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}
	// localFS := collective.NewFileSystemRemoteStorage(filepath.Join(storageDir,
	// 	localTxLogPath))

	return collective.SynchronizedKV(storageDir, string(password),
		remote, localKV, synchedPrefixes, rng)
}
