////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package utility

import (
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentCyclicVersion = 0

func StoreCyclicKey(kv versioned.KV, cy *cyclic.Int, key string) error {
	now := netTime.Now()

	data, err := cy.GobEncode()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentCyclicVersion,
		Timestamp: now,
		Data:      data,
	}

	return kv.Set(key, &obj)
}

func LoadCyclicKey(kv versioned.KV, key string) (*cyclic.Int, error) {
	vo, err := kv.Get(key, currentCyclicVersion)
	if err != nil {
		return nil, err
	}

	cy := &cyclic.Int{}

	return cy, cy.GobDecode(vo.Data)
}

// DeleteCyclicKey deletes a given cyclic key from storage
func DeleteCyclicKey(kv versioned.KV, key string) error {
	return kv.Delete(key, currentCyclicVersion)
}
