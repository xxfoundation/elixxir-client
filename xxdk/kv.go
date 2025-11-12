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
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/ekv/portable"
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
	return collective.LocalKV(password, localKV, rng)
}

// LocalKVWithKV creates a filesystem based KV backed by a custom key-value
// store that doesn't synchronize with a remote storage system.
func LocalKVWithKV(kv portable.GenericKeyValue, storageDir string, password []byte,
	rng *fastRNG.StreamGenerator) (versioned.KV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewKeyValueFilestore(kv, storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}
	return collective.LocalKV(password, localKV, rng)
}

// SynchronizedKV creates a filesystem based KV that synchronizes
// with a remote storage system.
func SynchronizedKV(storageDir, remoteStoragePathPrefix string, password []byte,
	remote collective.RemoteStore,
	synchedPrefixes []string,
	rng *fastRNG.StreamGenerator) (versioned.KV, error) {
	passwordStr := string(password)
	localKV, err := ekv.NewFilestore(storageDir, passwordStr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to create storage session")
	}

	return collective.SynchronizedKV(remoteStoragePathPrefix, password,
		remote, localKV, synchedPrefixes, rng)
}
