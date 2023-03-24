////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentCyclicVersion = 0

func StoreCyclicKey(kv *KV, cy *cyclic.Int, key string) error {
	now := netTime.Now()

	data, err := cy.GobEncode()
	if err != nil {
		return err
	}

	object := versioned.Object{
		Version:   currentCyclicVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(makeStorageKey(key), object.Marshal())
}

func LoadCyclicKey(kv *KV, key string) (*cyclic.Int, error) {
	data, err := kv.Get(makeStorageKey(key), currentCyclicVersion)
	if err != nil {
		return nil, err
	}

	cy := &cyclic.Int{}

	return cy, cy.GobDecode(data)
}

// DeleteCyclicKey deletes a given cyclic key from storage
func DeleteCyclicKey(kv *KV, key string) error {
	return kv.Delete(makeStorageKey(key), currentCyclicVersion)
}

func makeStorageKey(key string) string {
	return "e2eSession/" + key
}
