package e2e

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage/versioned"
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
	confirmStatus Confirmation

	// Value of the counter at which a rekey is triggered
	ttl uint32

	// Received Keys dirty bits
	// Each bit represents a single Key
	keyState *stateVector

	//mutex
	mux sync.RWMutex
}

// As this is serialized by json, any field that should be serialized
// must be exported
// Utility struct to write part of session data to disk
type SessionDisk struct {
	Params SessionParams

	//session type
	Type uint8

	// Underlying key
	BaseKey []byte
	// Own Private Key
	MyPrivKey []byte
	// Partner Public Key
	PartnerPubKey []byte

	//denotes if the other party has confirmed this key
	Confirmation uint8

	// Number of keys usable before rekey
	TTL uint32
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

	err := session.generate()
	if err != nil {
		return nil, err
	}

	err = session.save()
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Load session and state vector from kv and populate runtime fields
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

	if session.t == Receive {
		// register key fingerprints
		manager.ctx.fa.add(session.getUnusedKeys())
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

	obj := versioned.Object{
		Version:   currentSessionVersion,
		Timestamp: now,
		Data:      data,
	}

	return s.manager.ctx.kv.Set(key, &obj)
}

/*METHODS*/
// Remove all unused key fingerprints
// Delete this session and its key states from the storage
func (s *Session) Delete() {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.manager.ctx.fa.remove(s.getUnusedKeys())

	stateVectorKey := makeStateVectorKey(keyEKVPrefix, s.GetID())
	stateVectorErr := s.manager.ctx.kv.Delete(stateVectorKey)
	sessionKey := makeSessionKey(s.GetID())
	sessionErr := s.manager.ctx.kv.Delete(sessionKey)

	if stateVectorErr != nil && sessionErr != nil {
		jww.ERROR.Printf("Error deleting state vector with key %v: %v", stateVectorKey, stateVectorErr.Error())
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if sessionErr != nil {
		jww.ERROR.Panicf("Error deleting session with key %v: %v", sessionKey, sessionErr)
	} else if stateVectorErr != nil {
		jww.ERROR.Panicf("Error deleting state vector with key %v: %v", stateVectorKey, stateVectorErr.Error())
	}
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

	sd.Params = s.params
	sd.Type = uint8(s.t)
	sd.BaseKey = s.baseKey.Bytes()
	sd.MyPrivKey = s.myPrivKey.Bytes()
	sd.PartnerPubKey = s.partnerPubKey.Bytes()
	sd.Confirmed = s.confirmed
	sd.TTL = s.ttl

	return json.Marshal(&sd)
}

func (s *Session) unmarshal(b []byte) error {

	sd := SessionDisk{}

	err := json.Unmarshal(b, &sd)

	if err != nil {
		return err
	}

	grp := s.manager.ctx.grp

	s.params = sd.Params
	s.t = SessionType(sd.Type)
	s.baseKey = grp.NewIntFromBytes(sd.BaseKey)
	s.myPrivKey = grp.NewIntFromBytes(sd.MyPrivKey)
	s.partnerPubKey = grp.NewIntFromBytes(sd.PartnerPubKey)
	s.confirmed = sd.Confirmed
	s.ttl = sd.TTL

	statesKey := makeStateVectorKey(keyEKVPrefix, s.GetID())
	s.keyState, err = loadStateVector(s.manager.ctx, statesKey)
	if err != nil {
		return err
	}


	return nil
}

//key usage
// Pops the first unused key, skipping any which are denoted as used.
// will return if the remaining keys are designated as rekeys
func (s *Session) PopKey() (*Key, error) {
	if s.keyState.GetNumAvailable() <= uint32(s.params.NumRekeys) {
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
	if s.keyState.GetNumAvailable() == 0 {
		return RekeyEmpty
	} else if s.keyState.GetNumAvailable() <= uint32(s.params.NumRekeys) {
		return Empty
	} else if s.keyState.GetNumAvailable() <= s.keyState.GetNumKeys()-s.ttl {
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
func (s *Session) generate() error {
	grp := s.manager.ctx.grp

	//generate private key if it is not present
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

	// To generate the state vector key correctly,
	// basekey must be computed as the session ID is the hash of basekey
	var err error
	s.keyState, err = newStateVector(s.manager.ctx, makeStateVectorKey(keyEKVPrefix, s.GetID()), numKeys)
	if err != nil {
		return errors.WithMessage(err, "Failed key generation")
	}

	//register keys for reception if this is a reception session
	if s.t == Receive {
		//register keys
		s.manager.ctx.fa.add(s.getUnusedKeys())
	}

	return nil
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
