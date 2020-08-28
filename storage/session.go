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
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
)

// Session object, backed by encrypted filestore
type Session struct {
	kv  *versioned.KV
	mux sync.RWMutex

	regStatus RegistrationStatus

	//sub-stores
	e2e  *e2e.Store
	cmix *cmix.Store
	user *user.User

	loaded bool

}

// Initialize a new Session object
func Init(baseDir, password string) (*Session, error) {
	fs, err := ekv.NewFilestore(baseDir, password)
	var s *Session
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to create storage session")
	}

	s = &Session{
		kv:     versioned.NewKV(fs),
		loaded: false,
	}

	err = s.loadOrCreateRegStatus()
	if err != nil {
		return nil, errors.WithMessage(err,
			"Failed to load or create registration status")
	}

	return s, nil
}

// Creates new UserData in the session
func (s *Session) Create(uid *id.ID, salt []byte, rsaKey *rsa.PrivateKey,
	isPrecanned bool, cmixDHPrivKey, e2eDHPrivKey *cyclic.Int, cmixGrp,
	e2eGrp *cyclic.Group) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.loaded {
		return errors.New("Cannot create a session which already has one loaded")
	}

	var err error

	s.user, err = user.NewUser(s.kv, uid, salt, rsaKey, isPrecanned)
	if err != nil {
		return errors.WithMessage(err, "Failed to create Session due "+
			"to failed user creation")
	}

	s.cmix, err = cmix.NewStore(cmixGrp, s.kv, cmixDHPrivKey)
	if err != nil {
		return errors.WithMessage(err, "Failed to create Session due "+
			"to failed cmix keystore creation")
	}

	s.e2e, err = e2e.NewStore(e2eGrp, s.kv, e2eDHPrivKey)
	if err != nil {
		return errors.WithMessage(err, "Failed to create Session due "+
			"to failed e2e keystore creation")
	}

	s.loaded = true
	return nil
}

// Loads existing user data into the session
func (s *Session) Load() error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.loaded {
		return errors.New("Cannot load a session which already has one loaded")
	}

	var err error

	s.user, err = user.LoadUser(s.kv)
	if err != nil {
		return errors.WithMessage(err, "Failed to load Session due "+
			"to failure to load user")
	}

	s.cmix, err = cmix.LoadStore(s.kv)
	if err != nil {
		return errors.WithMessage(err, "Failed to load Session due "+
			"to failure to load cmix keystore")
	}

	s.e2e, err = e2e.LoadStore(s.kv)
	if err != nil {
		return errors.WithMessage(err, "Failed to load Session due "+
			"to failure to load e2e keystore")
	}

	s.loaded = true
	return nil
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
