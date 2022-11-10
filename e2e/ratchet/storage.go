////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"encoding/json"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v5/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/v5/storage/utility"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	currentStoreVersion = 0
	storeKey            = "Store"
)

// Load loads an extant ratchet from disk
func Load(kv *versioned.KV, myID *id.ID, grp *cyclic.Group,
	cyHandler session.CypherHandler, services Services, rng *fastRNG.StreamGenerator) (
	*Ratchet, error) {
	kv = kv.Prefix(packagePrefix)

	privKey, err := util.LoadCyclicKey(kv, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	pubKey, err := util.LoadCyclicKey(kv, pubKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	r := &Ratchet{
		managers: make(map[id.ID]partner.Manager),
		services: make(map[string]message.Processor),

		myID:                   myID,
		advertisedDHPrivateKey: privKey,
		advertisedDHPublicKey:  pubKey,

		kv: kv,

		cyHandler:  cyHandler,
		grp:        grp,
		rng:        rng,
		sInterface: services,
	}

	obj, err := kv.Get(storeKey, currentStoreVersion)
	if err != nil {
		return nil, err
	} else {
		err = r.unmarshal(obj.Data)
		if err != nil {
			return nil, err
		}
	}

	// add standard services
	if err = r.AddService(Silent, nil); err != nil {
		jww.FATAL.Panicf("Could not add standard %s "+
			"service: %+v", Silent, err)
	}
	if err = r.AddService(E2e, nil); err != nil {
		jww.FATAL.Panicf("Could not add standard %s "+
			"service: %+v", E2e, err)
	}

	return r, nil
}

func (r *Ratchet) save() error {
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

	return r.kv.Set(storeKey, &obj)
}

// ekv functions

func (r *Ratchet) marshal() ([]byte, error) {
	contacts := make([]id.ID, len(r.managers))

	index := 0
	for rid := range r.managers {
		contacts[index] = rid
		index++
	}

	err := util.StoreCyclicKey(r.kv, r.advertisedDHPrivateKey, privKeyKey)
	if err != nil {
		return nil, err
	}
	err = util.StoreCyclicKey(r.kv, r.advertisedDHPublicKey, pubKeyKey)
	if err != nil {
		return nil, err
	}

	return json.Marshal(&contacts)
}

func (r *Ratchet) unmarshal(b []byte) error {

	var contacts []id.ID

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for i := range contacts {
		//load the contact separately to ensure pointers do
		// not get swapped
		partnerID := (&contacts[i]).DeepCopy()
		// Load the relationship. The relationship handles
		// adding the fingerprints via the context object
		manager, err := partner.LoadManager(r.kv, r.myID, partnerID,
			r.cyHandler, r.grp, r.rng)
		if err != nil {
			jww.FATAL.Panicf("cannot load relationship for partner"+
				" %s: %s", partnerID, err.Error())
		}

		if !manager.PartnerId().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded manager with the wrong "+
				"partner ID: \n\t loaded: %s \n\t present: %s",
				partnerID, manager.PartnerId())
		}

		//add services for the manager
		r.add(manager)

		//assume
		r.managers[*partnerID] = manager
	}

	r.advertisedDHPrivateKey, err = util.LoadCyclicKey(r.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	r.advertisedDHPublicKey, err = util.LoadCyclicKey(r.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	return nil
}
