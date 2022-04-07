package ratchet

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
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

	r := &Ratchet{
		managers: make(map[partner.ManagerIdentity]*partner.Manager),
		services: make(map[string]message.Processor),

		defaultID: myID,

		kv: kv,

		cyHandler: cyHandler,
		grp:       grp,
		rng:       rng,
		sInteface: services,
	}

	obj, err := kv.Get(storeKey, currentStoreVersion)
	if err != nil {
		//try to load an old one
		obj, err = kv.Get(storeKey, 0)
		if err != nil {
			return nil, err
		} else {
			err = r.unmarshalOld(obj.Data)
			if err != nil {
				return nil, err
			}
		}

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

	return r.kv.Set(storeKey, currentStoreVersion, &obj)
}

// ekv functions
func (r *Ratchet) marshal() ([]byte, error) {
	contacts := make([]partner.ManagerIdentity, len(r.managers))

	index := 0
	for rid, _ := range r.managers {
		contacts[index] = rid
		index++
	}

	return json.Marshal(&contacts)
}

// In the event an old structure was loaded, unmarshal it and upgrade it
// todo: test this with some old kv data
func (r *Ratchet) unmarshalOld(b []byte) error {

	var contacts []id.ID

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for i := range contacts {
		//load the contact separately to ensure pointers do not get swapped
		partnerID := (&contacts[i]).DeepCopy()
		// Load the relationship. The relationship handles adding the fingerprints via the
		// context object
		manager, err := partner.LoadManager(r.kv, r.defaultID, partnerID,
			r.cyHandler, r.grp, r.rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %s: %s",
				partnerID, err.Error())
		}

		if !manager.GetPartnerID().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded a manager with the wrong partner "+
				"ID: \n\t loaded: %s \n\t present: %s",
				partnerID, manager.GetPartnerID())
		}

		//add services for the manager
		r.add(manager)

		//assume
		r.managers[partner.MakeManagerIdentity(partnerID, r.defaultID)] = manager
	}

	r.defaultDHPrivateKey, err = util.LoadCyclicKey(r.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	r.defaultDHPublicKey, err = util.LoadCyclicKey(r.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	return nil
}

func (r *Ratchet) unmarshal(b []byte) error {

	var contacts []partner.ManagerIdentity

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for i := range contacts {
		//load the contact separately to ensure pointers do not get swapped
		partnerID := contacts[i].GetPartner()
		myID := contacts[i].GetMe()
		// Load the relationship. The relationship handles adding the fingerprints via the
		// context object
		manager, err := partner.LoadManager(r.kv, myID, partnerID,
			r.cyHandler, r.grp, r.rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %s: %s",
				partnerID, err.Error())
		}

		if !manager.GetPartnerID().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded a manager with the wrong partner "+
				"ID: \n\t loaded: %s \n\t present: %s",
				partnerID, manager.GetPartnerID())
		}

		//add services for the manager
		r.add(manager)

		r.managers[contacts[i]] = manager
	}

	r.defaultDHPrivateKey, err = util.LoadCyclicKey(r.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	r.defaultDHPublicKey, err = util.LoadCyclicKey(r.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	return nil
}
