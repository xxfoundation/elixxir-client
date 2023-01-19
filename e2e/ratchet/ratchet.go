////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"sync"

	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

const (
	packagePrefix = "e2eSession"
	pubKeyKey     = "DhPubKey"
	privKeyKey    = "DhPrivKey"
)

var NoPartnerErrorStr = "No relationship with partner found"

type Ratchet struct {
	managers map[id.ID]partner.Manager
	mux      sync.RWMutex

	myID                   *id.ID
	advertisedDHPrivateKey *cyclic.Int
	advertisedDHPublicKey  *cyclic.Int

	grp       *cyclic.Group
	cyHandler session.CypherHandler
	rng       *fastRNG.StreamGenerator

	// services handler
	services    map[string]message.Processor
	sInterface  Services
	servicesMux sync.RWMutex

	kv *versioned.KV
}

// New creates a new store for the passed user ID and private key.
// The store can then be accessed by calling LoadStore.
// Does not create at a unique prefix, if multiple Ratchets are needed, make
// sure to add an uint prefix to the KV before instantiation.
func New(kv *versioned.KV, myID *id.ID, privKey *cyclic.Int,
	grp *cyclic.Group) error {

	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	r := &Ratchet{
		managers: make(map[id.ID]partner.Manager),
		services: make(map[string]message.Processor),

		myID:                   myID,
		advertisedDHPrivateKey: privKey,
		advertisedDHPublicKey:  pubKey,

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
func (r *Ratchet) AddPartner(partnerID *id.ID,
	partnerPubKey, myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey, sendParams,
	receiveParams session.Params) (partner.Manager, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	myID := r.myID

	myPubKey := diffieHellman.GeneratePublicKey(myPrivKey, r.grp)
	jww.INFO.Printf("Adding Partner %s:\n\tMy Public Key: %s"+
		"\n\tPartner Public Key: %s to %s",
		partnerID,
		myPubKey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0), myID)

	mid := *partnerID

	if _, ok := r.managers[mid]; ok {
		return nil, errors.New("Cannot overwrite existing partner")
	}
	m := partner.NewManager(r.kv, r.myID, partnerID, myPrivKey,
		partnerPubKey, mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, receiveParams, r.cyHandler, r.grp, r.rng)

	r.managers[mid] = m
	if err := r.save(); err != nil {
		jww.FATAL.Printf("Failed to add Partner %s: Save of store failed: %s",
			partnerID, err)
	}

	// Add services for the manager
	r.add(m)

	return m, nil
}

// GetPartner returns the partner per its ID, if it exists
func (r *Ratchet) GetPartner(partnerID *id.ID) (partner.Manager, error) {
	r.mux.RLock()
	defer r.mux.RUnlock()

	m, ok := r.managers[*partnerID]

	if !ok {
		jww.WARN.Printf("%s: %s", NoPartnerErrorStr, partnerID)
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// DeletePartner removes the associated contact from the E2E store
func (r *Ratchet) DeletePartner(partnerID *id.ID) error {
	m, ok := r.managers[*partnerID]
	if !ok {
		jww.WARN.Printf("%s: %s", NoPartnerErrorStr, partnerID)
		return errors.New(NoPartnerErrorStr)
	}

	if err := m.Delete(); err != nil {
		return errors.WithMessagef(err,
			"Could not remove partner %s from store",
			partnerID)
	}

	// Delete services
	r.delete(m)

	delete(r.managers, *partnerID)
	return r.save()

}

// GetAllPartnerIDs returns a list of all partner IDs that the user has
// an E2E relationship with.
func (r *Ratchet) GetAllPartnerIDs() []*id.ID {
	r.mux.RLock()
	defer r.mux.RUnlock()

	partnerIDs := make([]*id.ID, 0, len(r.managers))

	for _, m := range r.managers {
		partnerIDs = append(partnerIDs, m.PartnerId())
	}

	return partnerIDs
}

// GetDHPrivateKey returns the diffie hellman private key used
// to initially establish the ratchet.
func (r *Ratchet) GetDHPrivateKey() *cyclic.Int {
	return r.advertisedDHPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key used
// to initially establish the ratchet.
func (r *Ratchet) GetDHPublicKey() *cyclic.Int {
	return r.advertisedDHPublicKey
}
