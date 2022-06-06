///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/params"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
)

const (
	currentStoreVersion = 0
	packagePrefix       = "e2eSession"
	storeKey            = "Store"
	pubKeyKey           = "DhPubKey"
	privKeyKey          = "DhPrivKey"
	grpKey              = "Group"
	sidhPubKeyKey       = "SidhPubKey"
	sidhPrivKeyKey      = "SidhPrivKey"
)

var NoPartnerErrorStr = "No relationship with partner found"

type Store struct {
	managers map[id.ID]*Manager
	mux      sync.RWMutex

	dhPrivateKey *cyclic.Int
	dhPublicKey  *cyclic.Int
	grp          *cyclic.Group

	kv *versioned.KV

	*fingerprints

	*context

	e2eParams params.E2ESessionParams
}

func NewStore(grp *cyclic.Group, kv *versioned.KV, privKey *cyclic.Int,
	myID *id.ID, rng *fastRNG.StreamGenerator) (*Store, error) {
	// Generate public key
	pubKey := diffieHellman.GeneratePublicKey(privKey, grp)

	// Modify the prefix of the KV
	kv = kv.Prefix(packagePrefix)

	// Create new fingerprint map
	fingerprints := newFingerprints()

	s := &Store{
		managers: make(map[id.ID]*Manager),

		dhPrivateKey: privKey,
		dhPublicKey:  pubKey,
		grp:          grp,

		fingerprints: &fingerprints,

		kv: kv,

		context: &context{
			fa:   &fingerprints,
			grp:  grp,
			rng:  rng,
			myID: myID,
		},

		e2eParams: params.GetDefaultE2ESessionParams(),
	}

	err := util.StoreCyclicKey(kv, pubKey, pubKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e DH public key")
	}

	err = util.StoreCyclicKey(kv, privKey, privKeyKey)
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to store e2e DH private key")
	}

	err = util.StoreGroup(kv, grp, grpKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to store e2e group")
	}

	return s, s.save()
}

func LoadStore(kv *versioned.KV, myID *id.ID, rng *fastRNG.StreamGenerator) (*Store, error) {
	fingerprints := newFingerprints()
	kv = kv.Prefix(packagePrefix)

	grp, err := util.LoadGroup(kv, grpKey)
	if err != nil {
		return nil, err
	}

	s := &Store{
		managers: make(map[id.ID]*Manager),

		fingerprints: &fingerprints,

		kv:  kv,
		grp: grp,

		context: &context{
			fa:   &fingerprints,
			rng:  rng,
			myID: myID,
			grp:  grp,
		},

		e2eParams: params.GetDefaultE2ESessionParams(),
	}

	obj, err := kv.Get(storeKey, currentStoreVersion)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	s.context.grp = s.grp

	return s, nil
}

func (s *Store) save() error {
	now := netTime.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(storeKey, currentStoreVersion, &obj)
}

func (s *Store) AddPartner(partnerID *id.ID, partnerPubKey,
	myPrivKey *cyclic.Int, partnerSIDHPubKey *sidh.PublicKey,
	mySIDHPrivKey *sidh.PrivateKey,
	sendParams, receiveParams params.E2ESessionParams) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	jww.INFO.Printf("Adding Partner %s:\n\tMy Private Key: %s"+
		"\n\tPartner Public Key: %s",
		partnerID,
		myPrivKey.TextVerbose(16, 0),
		partnerPubKey.TextVerbose(16, 0))

	if _, ok := s.managers[*partnerID]; ok {
		return errors.New("Cannot overwrite existing partner")
	}

	m := newManager(s.context, s.kv, partnerID, myPrivKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		sendParams, receiveParams)

	s.managers[*partnerID] = m
	if err := s.save(); err != nil {
		jww.FATAL.Printf("Failed to add Partner %s: Save of store failed: %s",
			partnerID, err)
	}

	return nil
}

// DeletePartner removes the associated contact from the E2E store
func (s *Store) DeletePartner(partnerId *id.ID) error {
	m, ok := s.managers[*partnerId]
	if !ok {
		return errors.New(NoPartnerErrorStr)
	}

	if err := clearManager(m, s.kv); err != nil {
		return errors.WithMessagef(err, "Could not remove partner %s from store", partnerId)
	}

	delete(s.managers, *partnerId)
	return s.save()
}

func (s *Store) GetPartner(partnerID *id.ID) (*Manager, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m, ok := s.managers[*partnerID]

	if !ok {
		return nil, errors.New(NoPartnerErrorStr)
	}

	return m, nil
}

// GetPartnerContact find the partner with the given ID and assembles and
// returns a contact.Contact with their ID and DH key. An error is returned if
// no partner exists for the given ID.
func (s *Store) GetPartnerContact(partnerID *id.ID) (contact.Contact, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	// Get partner
	m, exists := s.managers[*partnerID]
	if !exists {
		return contact.Contact{}, errors.New(NoPartnerErrorStr)
	}

	// Assemble Contact
	c := contact.Contact{
		ID:       m.GetPartnerID(),
		DhPubKey: m.GetPartnerOriginPublicKey(),
	}

	return c, nil
}

// GetPartners returns a list of all partner IDs that the user has
// an E2E relationship with.
func (s *Store) GetPartners() []*id.ID {
	s.mux.RLock()
	defer s.mux.RUnlock()

	partnerIds := make([]*id.ID, 0, len(s.managers))

	for partnerId := range s.managers {
		pid := partnerId
		partnerIds = append(partnerIds, &pid)
	}

	return partnerIds
}

// PopKey pops a key for use based upon its fingerprint.
func (s *Store) PopKey(f format.Fingerprint) (*Key, bool) {
	return s.fingerprints.Pop(f)
}

// CheckKey checks that a key exists for the key fingerprint.
func (s *Store) CheckKey(f format.Fingerprint) bool {
	return s.fingerprints.Check(f)
}

// GetDHPrivateKey returns the diffie hellman private key.
func (s *Store) GetDHPrivateKey() *cyclic.Int {
	return s.dhPrivateKey
}

// GetDHPublicKey returns the diffie hellman public key.
func (s *Store) GetDHPublicKey() *cyclic.Int {
	return s.dhPublicKey
}

// GetGroup returns the cyclic group used for cMix.
func (s *Store) GetGroup() *cyclic.Group {
	return s.grp
}

// ekv functions

func (s *Store) marshal() ([]byte, error) {
	contacts := make([]id.ID, len(s.managers))

	index := 0
	for partnerID := range s.managers {
		contacts[index] = partnerID
		index++
	}

	return json.Marshal(&contacts)
}

func (s *Store) unmarshal(b []byte) error {

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
		manager, err := loadManager(s.context, s.kv, partnerID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load relationship for partner %s: %s",
				partnerID, err.Error())
		}

		if !manager.GetPartnerID().Cmp(partnerID) {
			jww.FATAL.Panicf("Loaded a manager with the wrong partner "+
				"ID: \n\t loaded: %s \n\t present: %s",
				partnerID, manager.GetPartnerID())
		}

		s.managers[*partnerID] = manager
	}

	s.dhPrivateKey, err = util.LoadCyclicKey(s.kv, privKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH private key")
	}

	s.dhPublicKey, err = util.LoadCyclicKey(s.kv, pubKeyKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH public key")
	}

	s.grp, err = util.LoadGroup(s.kv, grpKey)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to load e2e DH group")
	}

	return nil
}

// GetE2ESessionParams returns a copy of the session params object
func (s *Store) GetE2ESessionParams() params.E2ESessionParams {
	s.mux.RLock()
	defer s.mux.RUnlock()
	jww.DEBUG.Printf("Using Session Params: %s", s.e2eParams)
	return s.e2eParams
}

// SetE2ESessionParams overwrites the current session params
func (s *Store) SetE2ESessionParams(newParams params.E2ESessionParams) {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("Setting Session Params: %s", newParams)
	s.e2eParams = newParams
}

type fingerprints struct {
	toKey map[format.Fingerprint]*Key
	mux   sync.RWMutex
}

// newFingerprints creates a new fingerprints with an empty map.
func newFingerprints() fingerprints {
	return fingerprints{
		toKey: make(map[format.Fingerprint]*Key),
	}
}

// fingerprints adheres to the fingerprintAccess interface.

func (f *fingerprints) add(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		f.toKey[k.Fingerprint()] = k
		jww.TRACE.Printf("Added Key Fingerprint: %s",
			k.Fingerprint())
	}
}

func (f *fingerprints) remove(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		delete(f.toKey, k.Fingerprint())
	}
}

func (f *fingerprints) Check(fingerprint format.Fingerprint) bool {
	f.mux.RLock()
	defer f.mux.RUnlock()

	_, ok := f.toKey[fingerprint]
	return ok
}

func (f *fingerprints) Pop(fingerprint format.Fingerprint) (*Key, bool) {
	f.mux.Lock()
	defer f.mux.Unlock()
	key, ok := f.toKey[fingerprint]

	if !ok {
		return nil, false
	}

	delete(f.toKey, fingerprint)

	key.denoteUse()

	key.fp = &fingerprint

	return key, true
}
