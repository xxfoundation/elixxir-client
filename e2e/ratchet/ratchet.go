///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/network/message"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"sync"
)

const (
	packagePrefix = "e2eSession"
	pubKeyKey     = "DhPubKey"
	privKeyKey    = "DhPrivKey"
)

var NoPartnerErrorStr = "No relationship with partner found"

type Ratchet struct {
	managers map[relationshipIdentity]*partner.Manager
	mux      sync.RWMutex

	defaultID           *id.ID
	defaultDHPrivateKey *cyclic.Int
	defaultDHPublicKey  *cyclic.Int

	grp       *cyclic.Group
	cyHandler session.CypherHandler
	rng       *fastRNG.StreamGenerator

	//services handler
	services    map[string]message.Processor
	sInteface   Services
	servicesmux sync.RWMutex

	kv    *versioned.KV
	memKv *versioned.KV
}

// New creates a new store for the passed user id and private key.
// The store can then be accessed by calling LoadStore.
// Does not create at a unique prefix, if multiple Ratchets are needed, make
// sure to add a uint prefix to the KV before instantiation.
func New(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group) error {

	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	r := &Ratchet{
		managers: make(map[relationshipIdentity]*partner.Manager),
		services: make(map[string]message.Processor),

		defaultID:           myID,
		defaultDHPrivateKey: privKey,
		defaultDHPublicKey:  pubKey,

		kv: kv,

		grp: grp,
	}

	err := util.StoreCyclicKey(kv, pubKey, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to store e2e DH public key")
	}

	err = util.StoreCyclicKey(kv, privKey, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to store e2e DH private key")
	}

	return r.save()
}

// AddPartner adds a partner. Automatically creates both send and receive
// sessions using the passed cryptographic data and per the parameters sent
func (r *Ratchet) AddPartner(myID *id.ID, myPrivateKey *cyclic.Int, partnerID *id.ID,
	partnerPubKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey, sendParams,
	receiveParams session.Params, temporary bool) (*partner.Manager, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if myID == nil {
		myID = r.defaultID
	}

	if myPrivateKey == nil {
		myPrivateKey = r.defaultDHPrivateKey
	}

	jww.INFO.Printf("Adding Partner %s:\n\tMy Private Key: %s"+
		"\n\tPartner Public Key: %s to %s",
		partnerID,
		myPrivateKey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0), myID)

	rship := makeRelationshipIdentity(partnerID, myID)

	if _, ok := r.managers[rship]; ok {
		return nil, errors.New("Cannot overwrite existing partner")
	}

	//pass a memory kv if it is supposed to be temporary
	kv := r.kv
	if temporary {
		kv = r.memKv
	}

	m := partner.NewManager(kv, r.defaultID, partnerID, myPrivateKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, receiveParams, r.cyHandler, r.grp, r.rng)

	r.managers[rship] = m
	if err := r.save(); err != nil {
		jww.FATAL.Printf("Failed to add Partner %s: Save of store failed: %s",
			partnerID, err)
	}

	//add services for the manager
	r.add(m)

	return m, nil
}

// GetPartner returns the partner per its ID, if it exists
func (r *Ratchet) GetPartner(partnerID *id.ID, myID *id.ID) (*partner.Manager, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	if myID == nil {
		myID = r.defaultID
	}

	m, ok := r.managers[makeRelationshipIdentity(partnerID, myID)]

	if !ok {
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// DeletePartner removes the associated contact from the E2E store
func (r *Ratchet) DeletePartner(partnerId *id.ID, myID *id.ID) error {
	if myID == nil {
		myID = r.defaultID
	}

	rShip := makeRelationshipIdentity(partnerId, myID)
	m, ok := r.managers[rShip]
	if !ok {
		return errors.New(NoPartnerErrorStr)
	}

	if err := partner.ClearManager(m, r.kv); err != nil {
		return errors.WithMessagef(err, "Could not remove partner %s from store", partnerId)
	}

	//delete services
	r.delete(m)

	delete(r.managers, rShip)
	return r.save()

}

// GetAllPartnerIDs returns a list of all partner IDs that the user has
// an E2E relationship with.
func (r *Ratchet) GetAllPartnerIDs(myID *id.ID) []*id.ID {
	r.mux.RLock()
	defer r.mux.RUnlock()

	partnerIds := make([]*id.ID, 0, len(r.managers))

	for _, m := range r.managers {
		if m.GetMyID().Cmp(myID) {
			partnerIds = append(partnerIds, m.GetPartnerID())
		}

	}

	return partnerIds
}

// GetDHPrivateKey returns the diffie hellman private key.
func (r *Ratchet) GetDHPrivateKey() *cyclic.Int {
	return r.defaultDHPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key.
func (r *Ratchet) GetDHPublicKey() *cyclic.Int {
	return r.defaultDHPublicKey
}
