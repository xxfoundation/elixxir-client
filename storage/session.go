////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Session object definition

package storage

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/storage/cmix"
	"gitlab.com/elixxir/client/storage/conversation"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/storage/partition"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
)

// Number of rounds to store in the CheckedRound buffer
const checkRoundsMaxSize = 1000000 / 64

// Session object, backed by encrypted filestore
type Session struct {
	kv  *versioned.KV
	mux sync.RWMutex

	regStatus RegistrationStatus

	//sub-stores
	e2e              *e2e.Store
	cmix             *cmix.Store
	user             *user.User
	conversations    *conversation.Store
	partition        *partition.Store
	criticalMessages *utility.E2eMessageBuffer
	garbledMessages  *utility.CmixMessageBuffer
	checkedRounds    *utility.KnownRounds
}

// Initialize a new Session object
func initStore(baseDir, password string) (*Session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *Session
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create storage session")
	}

	s = &Session{
		kv: versioned.NewKV(fs),
	}

	return s, nil
}

// Creates new UserData in the session
func New(baseDir, password string, uid *id.ID, salt []byte, rsaKey *rsa.PrivateKey,
	isPrecanned bool, cmixDHPrivKey, e2eDHPrivKey *cyclic.Int, cmixGrp,
	e2eGrp *cyclic.Group) (*Session, error) {

	s, err := initStore(baseDir, password)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	err = s.newRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err,
			"Create new session")
	}

	s.user, err = user.NewUser(s.kv, uid, salt, rsaKey, isPrecanned)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.cmix, err = cmix.NewStore(cmixGrp, s.kv, cmixDHPrivKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.e2e, err = e2e.NewStore(e2eGrp, s.kv, e2eDHPrivKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.criticalMessages, err = utility.NewE2eMessageBuffer(s.kv, criticalMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.garbledMessages, err = utility.NewCmixMessageBuffer(s.kv, garbledMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.checkedRounds, err = utility.NewKnownRounds(s.kv, checkedRoundsKey, checkRoundsMaxSize)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create session")
	}

	s.conversations = conversation.NewStore(s.kv)
	s.partition = partition.New(s.kv)

	return s, nil
}

// Loads existing user data into the session
func Load(baseDir, password string) (*Session, error) {
	s, err := initStore(baseDir, password)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	err = s.loadRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.user, err = user.LoadUser(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.cmix, err = cmix.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.e2e, err = e2e.LoadStore(s.kv)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load Session")
	}

	s.criticalMessages, err = utility.LoadE2eMessageBuffer(s.kv, criticalMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load session")
	}

	s.garbledMessages, err = utility.LoadCmixMessageBuffer(s.kv, garbledMessagesKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load session")
	}

	s.checkedRounds, err = utility.LoadKnownRounds(s.kv, checkedRoundsKey, checkRoundsMaxSize)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load session")
	}

	s.conversations = conversation.NewStore(s.kv)
	s.partition = partition.New(s.kv)

	return s, nil
}

func (s *Session) User() *user.User {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.user
}

func (s *Session) Cmix() *cmix.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.cmix
}

func (s *Session) E2e() *e2e.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.e2e
}

func (s *Session) GetCriticalMessages() *utility.E2eMessageBuffer {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.criticalMessages
}

func (s *Session) GetGarbledMessages() *utility.CmixMessageBuffer {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.garbledMessages
}

func (s *Session) GetCheckedRounds() *utility.KnownRounds {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.checkedRounds
}

func (s *Session) Conversations() *conversation.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.conversations
}

func (s *Session) Partition() *partition.Store {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.partition
}

// Get an object from the session
func (s *Session) Get(key string) (*versioned.Object, error) {
	return s.kv.Get(key)
}

// Set a value in the session
func (s *Session) Set(key string, object *versioned.Object) error {
	return s.kv.Set(key, object)
}

// Delete a value in the session
func (s *Session) Delete(key string) error {
	return s.kv.Delete(key)
}

// Initializes a Session object wrapped around a MemStore object.
// FOR TESTING ONLY
func InitTestingSession(i interface{}) *Session {
	switch i.(type) {
	case *testing.T:
		break
	case *testing.M:
		break
	case *testing.B:
		break
	default:
		globals.Log.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", i)
	}

	store := make(ekv.Memstore)
	return &Session{kv: versioned.NewKV(store)}

}
