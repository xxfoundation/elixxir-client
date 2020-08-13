package key

import (
	"encoding/base64"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"sync"
)

type SessionID [32]byte

func (sid SessionID) String() string {
	return base64.StdEncoding.EncodeToString(sid[:])
}

type Session struct {
	//pointer to manager
	manager *Manager

	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	myPrivKey *cyclic.Int
	// Partner Public Key
	partnerPubKey *cyclic.Int

	//denotes if the other party has confirmed this key
	confirmed bool

	// Value of the counter at which a rekey is triggered
	ttl uint16

	// Total number of Keys
	numKeys uint32
	// Total number of Rekey keys
	numReKeys uint16

	// Received Keys dirty bits
	// Each bit represents a single Key
	KeyState []*uint64
	// Received ReKeys dirty bits
	// Each bit represents a single ReKey
	ReKeyState []*uint64

	// Keys
	keys    []*Key
	lastKey uint32

	// ReKeys
	reKeys    []*Key
	lastReKey uint32

	//mutex
	mux sync.RWMutex
}

type SessionDisk struct {
	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	myPrivKey *cyclic.Int
	// Partner Public Key
	partnerPubKey *cyclic.Int

	// Received Keys dirty bits
	// Each bit represents a single Key
	KeyState []*uint64
	// Received ReKeys dirty bits
	// Each bit represents a single ReKey
	ReKeyState []*uint64

	//denotes if the other party has confirmed this key
	confirmed bool

	//position of the earliest unused key
	lastKey uint32

	//position of the earliest unused key
	lastReKey uint32
}

//Generator which creates all keys and structures
func (s *Session) NewSession(myPrivKey *cyclic.Int, partnerPubKey *cyclic.Int, manager *Manager) (*Session, error) {
	return nil, nil
}

//Gets the manager used to

//Gets the base key.
func (s *Session) GetBaseKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.baseKey.DeepCopy()
}

func (s *Session) GetMyPrivKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.myPrivKey.DeepCopy()
}

func (s *Session) GetPartnerPubKey() *cyclic.Int {
	// no lock is needed because this cannot be edited
	return s.partnerPubKey.DeepCopy()
}

//Blake2B hash of base key used for storage
func (s *Session) GetID() SessionID {
	sid := SessionID{}
	h, _ := hash.NewCMixHash()
	h.Write(s.baseKey.Bytes())
	copy(sid[:], h.Sum(nil))
	return sid
}

//ekv functions
func (s *Session) Marshal() ([]byte, error) { return nil, nil }
func (s *Session) Unmarshal([]byte) error   { return nil }

//key usage
// Pops the first unused key, skipping any which are denoted as used
func (s *Session) PopKey() (*keyStore.E2EKey, error) { return nil, nil }

// Pops the first unused rekey, skipping any which are denoted as used
func (s *Session) PopReKey() (*keyStore.E2EKey, error) { return nil, nil }

// denotes the passed key as used
func (s *Session) UseKey(*keyStore.E2EKey) error { return nil }

// denotes the passed rekey as used
func (s *Session) UseRekey(*keyStore.E2EKey) error { return nil }

// returns the state of the keyblob, which denotes if the Session is active,
// functional but in need of a rekey, empty of send key, or empty of rekeys
func (s *Session) Status() Status { return 0 }

// Sets the confirm bool. this is set when the partner is certain to share the
// session. It should be called immediately for receive keys and only on rekey
// confirmation for send keys. Confirmation can only be made by the sessionBuffer
// because it is used to keep track of active sessions for rekey as well
func (s *Session) confirm() {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.confirmed = true
}

// checks if the session has been confirmed
func (s *Session) IsConfirmed() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.confirmed
}

/*PRIVATE*/
