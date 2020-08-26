package key

import (
	"encoding/json"
	"errors"
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

	//denotes if the other party has confirmed this key
	confirmed bool

	// Value of the counter at which a rekey is triggered
	ttl uint32

	// Received Keys dirty bits
	// Each bit represents a single Key
	keyState *stateVector

	//mutex
	mux sync.RWMutex
}

type SessionDisk struct {
	params SessionParams

	//session type
	t uint8

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
	}

	session.generate()

	err := session.save()
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

	return &session, nil
}

func (s *Session) save() error {
	key := makeSessionKey(s.GetID())

	now := time.Now()

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

	s.manager.ctx.fa.remove(s.getUnusedKeys())

	return s.manager.ctx.kv.Delete(makeSessionKey(s.GetID()))
}

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
	// no lock is needed because this cannot be edited
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
	sd.t = uint8(s.t)
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
	s.t = SessionType(sd.t)
	s.baseKey = grp.NewIntFromBytes(sd.BaseKey)
	s.myPrivKey = grp.NewIntFromBytes(sd.MyPrivKey)
	s.partnerPubKey = grp.NewIntFromBytes(sd.PartnerPubKey)
	s.confirmed = sd.Confirmed

	sid := s.GetID()

	s.keyState, err = loadStateVector(s.manager.ctx, makeStateVectorKey("keyStates", sid))
	if err != nil {
		return err
	}

	if s.t == Receive {
		//register keys
		s.manager.ctx.fa.add(s.getUnusedKeys())
	}

	return nil
}

//key usage
// Pops the first unused key, skipping any which are denoted as used.
// will return if the remaining keys are designated as rekeys
func (s *Session) PopKey() (*Key, error) {
	if s.keyState.numkeys-s.keyState.numAvailable <= uint32(s.params.NumRekeys) {
		return nil, errors.New("no more keys left, remaining reserved " +
			"for rekey")
	}
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newKey(s, keyNum), nil
}

func (s *Session) PopReKey() (*Key, error) {
	keyNum, err := s.keyState.Next()
	if err != nil {
		return nil, err
	}

	return newKey(s, keyNum), nil
}

// returns the state of the session, which denotes if the Session is active,
// functional but in need of a rekey, empty of send key, or empty of rekeys
func (s *Session) Status() Status {
	if s.keyState.numkeys-s.keyState.numAvailable <= uint32(s.params.NumRekeys) {
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

// generates keys from the base data stored in the session object.
// myPrivKey will be generated if not present
func (s *Session) generate() {
	grp := s.manager.ctx.grp

	//generate public key if it is not present
	if s.myPrivKey == nil {
		s.myPrivKey = dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp,
			csprng.NewSystemRNG())
	}

	// compute the base key
	s.baseKey = dh.GenerateSessionKey(s.myPrivKey, s.partnerPubKey, grp)

	//generate ttl and keying info
	keysTTL, numKeys := e2e.GenerateKeyTTL(s.baseKey.GetLargeInt(),
		s.params.MinKeys, s.params.MaxKeys, s.params.TTLParams)

	//ensure that enough keys are remaining to rekey
	if numKeys-uint32(keysTTL) < uint32(s.params.NumRekeys) {
		numKeys = uint32(keysTTL + s.params.NumRekeys)
	}

	s.ttl = uint32(keysTTL)

	//create the new state vectors. This will cause disk operations storing them

	s.keyState = newStateVector(s.manager.ctx, keyEKVPrefix, numKeys)

	//register keys for reception if this is a reception session
	if s.t == Receive {
		//register keys
		s.manager.ctx.fa.add(s.getUnusedKeys())
	}
}

//returns key objects for all unused keys
func (s *Session) getUnusedKeys() []*Key {
	keyNums := s.keyState.GetUnusedKeyNums()

	keys := make([]*Key, len(keyNums))
	for i, keyNum := range keyNums {
		keys[i] = newKey(s, keyNum)
	}

	return keys
}
