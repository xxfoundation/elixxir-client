///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const prefix = "cmix"
const currentStoreVersion = 0
const (
	storeKey = "KeyStore"
	grpKey   = "GroupKey"
)

// Add adds the key for a round to the cMix storage object. Saves the updated
// list of nodes and the key to disk.
func (r *registrar) add(nid *id.ID, k *cyclic.Int,
	validUntil uint64, keyId []byte) {
	r.mux.Lock()
	defer r.mux.Unlock()

	nodeKey := newKey(r.kv, k, nid, validUntil, keyId)

	r.nodes[*nid] = nodeKey
	if err := r.save(); err != nil {
		jww.FATAL.Panicf("Failed to save nodeKey list for %s: %+v", nid, err)
	}
}

// Remove removes a nodes key from the nodes map and saves.
func (r *registrar) remove(nid *id.ID) {
	r.mux.Lock()
	defer r.mux.Unlock()

	nodeKey, ok := r.nodes[*nid]
	if !ok {
		return
	}

	nodeKey.delete(r.kv, nid)

	delete(r.nodes, *nid)

	if err := r.save(); err != nil {
		jww.FATAL.Panicf("Failed to make nodeKey for %s: %+v", nid, err)
	}
}

// save stores the cMix store.
func (r *registrar) save() error {
	now := netTime.Now()

	data, err := r.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return r.kv.Set(storeKey, currentKeyVersion, &obj)
}

// marshal builds a byte representation of the registrar.
func (r *registrar) marshal() ([]byte, error) {
	nodes := make([]id.ID, len(r.nodes))

	index := 0
	for nid := range r.nodes {
		nodes[index] = nid
		index++
	}

	return json.Marshal(&nodes)
}

// unmarshal restores the data for a registrar from the byte representation of
// the registrar.
func (r *registrar) unmarshal(b []byte) error {
	var nodes []id.ID

	err := json.Unmarshal(b, &nodes)
	if err != nil {
		return err
	}

	for _, nid := range nodes {
		k, err := loadKey(r.kv, &nid)
		if err != nil {
			return errors.WithMessagef(
				err, "could not load nodes key for %s", &nid)
		}
		r.nodes[nid] = k
	}

	return nil
}
