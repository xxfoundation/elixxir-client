package key

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"sync"
	"time"
)

const currentSessionVersion = 0
const keyEKVPrefix = "KEY"
const reKeyEKVPrefix = "REKEY"

type Session struct {
	//pointer to manager
	manager *Manager
	//params
	params SessionParams

	//type
	t SessionType

	// Underlying key
	baseKey *cyclic.Int
	// Own Private Key
	myPrivKey *cyclic.Int
	// Partner Public Key
	partnerPubKey *cyclic.Int

	//denotes if keys have been generated before
	generated bool

	//denotes if the other party has confirmed this key
	confirmed bool

	// Value of the counter at which a rekey is triggered
	ttl uint32

	// Received Keys dirty bits
	// Each bit represents a single Key
	keyState *stateVector
	// Received ReKeys dirty bits
	// Each bit represents a single ReKey
	reKeyState *stateVector

	//mutex
	mux sync.RWMutex
}

type SessionDisk struct {
	params SessionParams

	// Underlying key
	BaseKey []byte
	// Own Private Key
	MyPrivKey []byte
	// Partner Public Key
	PartnerPubKey []byte

	//denotes if the other party has confirmed this key
	Confirmed bool
}

/*CONSTRUCTORS*/
//Generator which creates all keys and structures
func newSession(manager *Manager, myPrivKey *cyclic.Int, partnerPubKey *cyclic.Int, params SessionParams, t SessionType) (*Session, error) {
	session := &Session{
		params:        params,
		manager:       manager,
		t:             t,
		myPrivKey:     myPrivKey,
		partnerPubKey: partnerPubKey,
		confirmed:     t == Receive,
		generated:     false,
	}

	err := session.generateKeys()
	if err != nil {
		return nil, err
	}

	err = session.save()
	if err != nil {
		return nil, err
	}

	return session, nil
}

//Generator which creates all keys and structures
func loadSession(manager *Manager, key string) (*Session, error) {

	session := Session{
		manager: manager,
	}

	obj, err := manager.ctx.kv.Get(key)
	if err != nil {
		return nil, err
	}

	err = session.unmarshal(obj.Data)
	if err != nil {
		return nil, err
	}

	session.generated = true

	return &session, nil
}

func (s *Session) save() error {
	key := makeSessionKey(s.GetID())

	now, err := time.Now().MarshalText()
	if err != nil {
		return err
	}

	data, err := s.marshal()
	if err != nil {
		return err
	}

	obj := storage.VersionedObject{
		Version:   currentSessionVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.manager.ctx.kv.Set(key, &obj)
}

/*METHODS*/
func (s *Session) Delete() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	//s.manager.ctx.fa.remove(s.keys)
	//s.manager.ctx.fa.remove(s.reKeys)

	return s.manager.ctx.kv.Delete(makeSessionKey(s.GetID()))
}

//Gets the base key.
func (s *Session) GetBaseKey() *cyclic.Int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	// no lock is needed because this cannot be edited
	return s.baseKey.DeepCopy()
}

func (s *Session) GetMyPrivKey() *cyclic.Int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	// no lock is needed because this cannot be edited
	return s.myPrivKey.DeepCopy()
}

func (s *Session) GetPartnerPubKey() *cyclic.Int {
	s.mux.RLock()
	defer s.mux.RUnlock()
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
func (s *Session) marshal() ([]byte, error) {
	sd := SessionDisk{}

	sd.params = s.params
	sd.BaseKey = s.baseKey.Bytes()
	sd.MyPrivKey = s.myPrivKey.Bytes()
	sd.PartnerPubKey = s.partnerPubKey.Bytes()
	sd.Confirmed = s.confirmed

	return json.Marshal(&sd)
}

func (s *Session) unmarshal(b []byte) error {

	sd := SessionDisk{}

	err := json.Unmarshal(b, &sd)

	if err != nil {
		return err
	}

	grp := s.manager.ctx.grp

	s.params = sd.params
	s.baseKey = grp.NewIntFromBytes(sd.BaseKey)
	s.myPrivKey = grp.NewIntFromBytes(sd.MyPrivKey)
	s.partnerPubKey = grp.NewIntFromBytes(sd.PartnerPubKey)
	s.confirmed = sd.Confirmed

	sid := s.GetID()

	s.keyState, err = loadStateVector(s.manager.ctx, makeStateVectorKey("keyStates", sid))
	if err != nil {
		return err
	}

	s.reKeyState, err = loadStateVector(s.manager.ctx, makeStateVectorKey("reKeyStates", sid))
	if err != nil {
		return err
	}

	return s.generateKeys()
}

//key usage
// Pops the first unused key, skipping any which are denoted as used. The status
// is returned to check if a rekey is nessessary
func (s *Session) PopKey() (*Key, error) {
	/*keynum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}*/

	return nil, nil
}

// Pops the first unused rekey, skipping any which are denoted as used
func (s *Session) PopReKey() (*Key, error) {
	/*keynum, err := s.reKeyState.Next()
	if err != nil {
		return nil, err
	}*/
	return nil, nil
}

// returns the state of the session, which denotes if the Session is active,
// functional but in need of a rekey, empty of send key, or empty of rekeys
func (s *Session) Status() Status {
	s.mux.RLock()
	defer s.mux.RUnlock()
	if s.reKeyState.GetNumKeys() == 0 {
		return RekeyEmpty
	} else if s.keyState.GetNumKeys() == 0 {
		return Empty
	} else if s.keyState.GetNumKeys() >= s.ttl {
		return RekeyNeeded
	} else {
		return Active
	}
}

// returns the state of the session, which denotes if the Session is active,
// functional but in need of a rekey, empty of send key, or empty of rekeys
func (s *Session) IsReKeyNeeded() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.keyState.GetNumAvailable() == s.ttl
}

// checks if the session has been confirmed
func (s *Session) IsConfirmed() bool {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.confirmed
}

/*PRIVATE*/

// Sets the confirm bool. this is set when the partner is certain to share the
// session. It should be called immediately for receive keys and only on rekey
// confirmation for send keys. Confirmation can only be made by the sessionBuffer
// because it is used to keep track of active sessions for rekey as well
func (s *Session) confirm() error {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.confirmed = true
	return s.save()
}

func (s *Session) useKey(keynum uint32) error {
	return s.keyState.Use(keynum)
}

func (s *Session) useReKey(keynum uint32) error {
	return s.reKeyState.Use(keynum)
}

// generates keys from the base data stored in the session object.
// required fields: partnerPubKey, manager
// myPrivKey, baseKey, keyState, and ReKeyState will be
// created/calculated if not present
// if keyState is not present lastKey will be ignored and set to zero
// if ReKeyState is not present lastReKey will be ignored and set to zero
func (s *Session) generateKeys() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	//check required fields
	if s.partnerPubKey == nil {
		return errors.New("Session must have a partner public key")
	}

	if s.manager == nil {
		return errors.New("Session must have a manager")
	}

	//generate optional fields if not present
	grp := s.manager.ctx.grp
	if s.myPrivKey == nil {
		rng := csprng.NewSystemRNG()
		s.myPrivKey = dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	}

	if s.baseKey == nil {
		s.baseKey = dh.GenerateSessionKey(s.myPrivKey, s.partnerPubKey, grp)
	}

	// generate key definitions if this is the first instantiation of the
	// session
	if !s.generated {
		//generate ttl and keying info
		keysTTL, numKeys := e2e.GenerateKeyTTL(s.baseKey.GetLargeInt(),
			s.params.MinKeys, s.params.MaxKeys, s.params.TTLParams)

		s.ttl = uint32(keysTTL)

		s.keyState = newStateVector(s.manager.ctx, keyEKVPrefix, numKeys)
		s.reKeyState = newStateVector(s.manager.ctx, reKeyEKVPrefix, uint32(s.params.NumRekeys))
	}

	// add the key and rekey fingerprints to the fingerprint map if in receiving
	// mode
	if s.t == Receive {
		//generate key
		keyNums := s.keyState.GetUnusedKeyNums()

		keys := make([]*Key, len(keyNums))
		for i, keyNum := range keyNums {
			keys[i] = newKey(s, parse.E2E, keyNum)
		}

		//register keys
		s.manager.ctx.fa.add(keys)

		//generate rekeys
		reKeyNums := s.reKeyState.GetUnusedKeyNums()
		rekeys := make([]*Key, len(reKeyNums))
		for i, rekeyNum := range reKeyNums {
			rekeys[i] = newKey(s, parse.Rekey, rekeyNum)
		}

		//register rekeys
		s.manager.ctx.fa.add(rekeys)
	}

	s.generated = true

	return nil
}