package e2e

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

const storeKey = "e2eKeyStore"
const currentStoreVersion = 0

type Store struct {
	managers map[id.ID]*Manager
	mux      sync.RWMutex

	fingerprints

	context
}

func NewStore(grp *cyclic.Group, kv *versioned.KV) *Store {
	fingerprints := newFingerprints()
	return &Store{
		managers:     make(map[id.ID]*Manager),
		fingerprints: fingerprints,
		context: context{
			fa:  &fingerprints,
			grp: grp,
			kv:  kv,
		},
	}

}

func LoadStore(grp *cyclic.Group, kv *versioned.KV) (*Store, error) {
	s := NewStore(grp, kv)

	obj, err := kv.Get(storeKey)
	if err != nil {
		return nil, err
	}

	err = s.unmarshal(obj.Data)

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) save() error {
	now := time.Now()

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := versioned.Object{
		Version:   currentStoreVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.kv.Set(storeKey, &obj)
}

func (s *Store) AddPartner(partnerID *id.ID, myPrivKey *cyclic.Int,
	partnerPubKey *cyclic.Int, sendParams, receiveParams SessionParams) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	m, err := newManager(&s.context, partnerID, myPrivKey, partnerPubKey, sendParams, receiveParams)

	if err != nil {
		return err
	}

	s.managers[*partnerID] = m

	return s.save()
}

func (s *Store) GetPartner(partnerID *id.ID) (*Manager, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	m, ok := s.managers[*partnerID]

	if !ok {
		return nil, errors.New("Cound not find manager for partner")
	}

	return m, nil
}

//ekv functions
func (s *Store) marshal() ([]byte, error) {

	contacts := make([]id.ID, len(s.managers))

	index := 0
	for partnerID := range s.managers {
		contacts[index] = partnerID
	}

	return json.Marshal(&contacts)
}

func (s *Store) unmarshal(b []byte) error {

	var contacts []id.ID

	err := json.Unmarshal(b, &contacts)

	if err != nil {
		return err
	}

	for _, partnerID := range contacts {
		// load the manager. Manager handles adding the fingerprints via the
		// context object
		manager, err := loadManager(&s.context, &partnerID)
		if err != nil {
			jww.FATAL.Panicf("Failed to load manager for partner %s: %s", &partnerID, err.Error())
		}

		s.managers[partnerID] = manager
	}
	return nil
}

type fingerprints struct {
	toKey map[format.Fingerprint]*Key
	mux   sync.RWMutex
}

func newFingerprints() fingerprints {
	return fingerprints{
		toKey: make(map[format.Fingerprint]*Key),
	}
}

//fingerprint adhere to the fingerprintAccess interface
func (f *fingerprints) add(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		f.toKey[k.Fingerprint()] = k
	}
}

func (f *fingerprints) remove(keys []*Key) {
	f.mux.Lock()
	defer f.mux.Unlock()

	for _, k := range keys {
		delete(f.toKey, k.Fingerprint())
	}
}

func (f *fingerprints) Pop(fingerprint format.Fingerprint) (*Key, error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	key, ok := f.toKey[fingerprint]

	if !ok {
		return nil, errors.New("Key could not be found")
	}

	delete(f.toKey, fingerprint)

	err := key.denoteUse()

	if err != nil {
		return nil, err
	}

	key.fp = &fingerprint

	return key, nil
}

