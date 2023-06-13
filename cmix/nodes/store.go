////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"encoding/base64"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	prefix                 = "cmix"
	currentStoreMapVersion = 0
	storeMapName           = "KeyMap"
)

func (r *registrar) loadStore() {
	// data is loaded due to the map update callback being called by ListenOnRemoteMap
	err := r.remote.ListenOnRemoteMap(storeMapName, currentStoreMapVersion, r.mapUpdate, false)

	if err != nil {
		jww.FATAL.Panicf("Registration with remote failed: %+v", err)
	}
}

func (r *registrar) mapUpdate(edits map[string]versioned.ElementEdit) {
	r.mux.Lock()
	defer r.mux.Unlock()

	for element, edit := range edits {
		nID := &id.ID{}

		if _, err := base64.StdEncoding.Decode(nID[:], []byte(element)); err != nil {
			jww.WARN.Printf("Failed to unmarshal key name in node key "+
				"storage on remote update op %s, skipping keyName: %s: %+v",
				edit.Operation, element, err)
			continue
		}

		if edit.Operation == versioned.Deleted {
			delete(r.nodes, *nID)
		} else {
			k := &key{}
			if err := k.unmarshal(edit.NewElement.Data); err != nil {
				jww.WARN.Printf("Failed to unmarshal node key "+
					" in key storage on remote update, skipping keyName: %s: %+v",
					nID, err)
				continue
			}
			r.nodes[*nID] = k
		}
	}
}

// Add adds the key for a round to the cMix storage object. Saves the updated
// list of nodes and the key to disk.
func (r *registrar) add(nid *id.ID, k *cyclic.Int,
	validUntil uint64, keyId []byte) {
	r.mux.Lock()
	defer r.mux.Unlock()

	nodeKey := newKey(k, validUntil, keyId)

	nodeKeyBytes, err := nodeKey.marshal()
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal new nodeKey %s: %+v", nid, err)
	}

	elementName, err := nid.MarshalText()
	if err != nil {
		jww.FATAL.Panicf("Failed to marshal test node id %s: %+v", nid, err)
	}

	err = r.remote.StoreMapElement(storeMapName, string(elementName), &versioned.Object{
		Version:   currentStoreMapVersion,
		Timestamp: netTime.Now(),
		Data:      nodeKeyBytes,
	}, currentStoreMapVersion)
	if err != nil {
		jww.FATAL.Panicf("Failed to store new nodeKey %s: %+v", nid, err)
	}

	r.nodes[*nid] = nodeKey
}

// Remove removes a nodes key from the nodes map and saves.
func (r *registrar) remove(nid *id.ID) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if _, ok := r.nodes[*nid]; !ok {
		return
	}

	elementName := string(nid.Marshal())

	if _, err := r.remote.DeleteMapElement(storeMapName, elementName,
		currentStoreMapVersion); err != nil {
		jww.FATAL.Panicf("Failed to delete nodeKey %s: %+v", nid, err)
	}

	delete(r.nodes, *nid)
}
