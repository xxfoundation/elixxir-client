////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"math/rand"
	"sync"
	"time"
)

// Errors
var ErrQuery = errors.New("element not in map")

// Interface for User Session operations
type Session interface {
	GetCurrentUser() (currentUser *User)
	GetKeys() []NodeKeys
	GetPrivateKey() *signature.DSAPrivateKey
	GetPublicKey() *signature.DSAPublicKey
	GetGroup() *cyclic.Group
	GetLastMessageID() string
	SetLastMessageID(id string)
	StoreSession() error
	Immolate() error
	UpsertMap(key string, element interface{}) error
	QueryMap(key string) (interface{}, error)
	DeleteMap(key string) error
	GetKeyStore() *keyStore.KeyStore
	GetRekeyManager() *keyStore.RekeyManager
	GetSwitchboard() *switchboard.Switchboard
	GetQuitChan() chan bool
	LockStorage()
	UnlockStorage()
	GetSessionData() ([]byte, error)
}

type NodeKeys struct {
	TransmissionKey *cyclic.Int
	ReceptionKey    *cyclic.Int
}

// Creates a new Session interface for registration
func NewSession(store globals.Storage,
	u *User, nk []NodeKeys,
	publicKey *signature.DSAPublicKey,
	privateKey *signature.DSAPrivateKey,
	grp *cyclic.Group) Session {

	// With an underlying Session data structure
	return Session(&SessionObj{
		CurrentUser:         u,
		Keys:                nk,
		PrivateKey:          privateKey,
		PublicKey:           publicKey,
		Grp:                 grp,
		InterfaceMap:        make(map[string]interface{}),
		KeyMaps:             keyStore.NewStore(),
		RekeyManager:        keyStore.NewRekeyManager(),
		store:               store,
		listeners:           switchboard.NewSwitchboard(),
		quitReceptionRunner: make(chan bool),
	})
}

func LoadSession(store globals.Storage,
	UID *id.User) (Session, error) {
	if store == nil {
		err := errors.New("LoadSession: Local Storage not avalible")
		return nil, err
	}

	rand.Seed(time.Now().UnixNano())

	sessionGob := store.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Write(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := SessionObj{}

	err := dec.Decode(&session)

	if err != nil {
		err = errors.New(fmt.Sprintf("LoadSession: unable to load session: %s", err.Error()))
		return nil, err
	}

	// FIXME We got a nil pointer dereference on the next lines but I haven't
	// been able to reproduce it. We should investigate why either of these
	// could be nil at this point. If you manage to reproduce the dereference
	// and you have the time, please try to figure out what's going on.
	// I suspect the client was loading some sort of malformed session gob,
	// and we need to fail faster in the case that a malformed gob got loaded.
	if session.CurrentUser.User == nil && UID == nil {
		jww.ERROR.Panic("Dereferencing nil session.CurrentUser.User AND UID")
	} else if session.CurrentUser.User == nil {
		jww.ERROR.Panic("Dereferencing nil session.CurrentUser.User")
	} else if UID == nil {
		jww.ERROR.Panic("Dereferencing nil param UID")
	}

	// Line of the actual crash
	if *session.CurrentUser.User != *UID {
		err = errors.New(fmt.Sprintf(
			"LoadSession: loaded incorrect "+
				"user; Expected: %q; Received: %q",
			*session.CurrentUser.User, *UID))
		return nil, err
	}

	// Reconstruct Key maps
	session.KeyMaps.ReconstructKeys(session.Grp, UID)

	// Create switchboard
	session.listeners = switchboard.NewSwitchboard()
	// Create quit channel for reception runner
	session.quitReceptionRunner = make(chan bool)

	// Set storage pointer
	session.store = store
	return &session, nil
}

// Struct holding relevant session data
type SessionObj struct {
	// Currently authenticated user
	CurrentUser *User

	Keys       []NodeKeys
	PrivateKey *signature.DSAPrivateKey
	PublicKey  *signature.DSAPublicKey
	Grp        *cyclic.Group

	// Last received message ID. Check messages after this on the gateway.
	LastMessageID string

	//Interface map for random data storage
	InterfaceMap map[string]interface{}

	// E2E KeyStore
	KeyMaps *keyStore.KeyStore

	// Rekey Manager
	RekeyManager *keyStore.RekeyManager

	// Non exported fields (not GOB encoded/decoded)
	// Local pointer to storage of this session
	store globals.Storage

	// Switchboard
	listeners *switchboard.Switchboard

	// Quit channel for message reception runner
	quitReceptionRunner chan bool

	lock sync.Mutex
}

func (s *SessionObj) GetLastMessageID() string {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.LastMessageID
}

func (s *SessionObj) SetLastMessageID(id string) {
	s.LockStorage()
	s.LastMessageID = id
	s.UnlockStorage()
}

func (s *SessionObj) GetKeys() []NodeKeys {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.Keys
}

func (s *SessionObj) GetPrivateKey() *signature.DSAPrivateKey {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.PrivateKey
}

func (s *SessionObj) GetPublicKey() *signature.DSAPublicKey {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.PublicKey
}

func (s *SessionObj) GetGroup() *cyclic.Group {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.Grp
}

// Return a copy of the current user
func (s *SessionObj) GetCurrentUser() (currentUser *User) {
	// This is where it deadlocks
	s.LockStorage()
	defer s.UnlockStorage()
	if s.CurrentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			User: s.CurrentUser.User,
			Nick: s.CurrentUser.Nick,
		}
	}
	return currentUser
}

func (s *SessionObj) storeSession() error {

	if s.store == nil {
		err := errors.New("StoreSession: Local Storage not available")
		return err
	}

	sessionData, err := s.getSessionData()
	err = s.store.Save(sessionData)
	if err != nil {
		return err
	}

	if err != nil {
		err = errors.New(fmt.Sprintf("StoreSession: Could not save the encoded user"+
			" session: %s", err.Error()))
		return err
	}

	return nil

}

func (s *SessionObj) StoreSession() error {
	s.LockStorage()
	err := s.storeSession()
	s.UnlockStorage()
	return err
}

// Immolate scrubs all cryptographic data from ram and logs out
// the ram overwriting can be improved
func (s *SessionObj) Immolate() error {
	s.LockStorage()
	if s == nil {
		err := errors.New("immolate: Cannot immolate that which has no life")
		return err
	}

	// clear data stored in session
	// Warning: be careful about immolating the memory backing the user ID
	// because that may alias a key in the user maps
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = ""

	s.UnlockStorage()

	return nil
}

//Upserts an element into the interface map and saves the session object
func (s *SessionObj) UpsertMap(key string, element interface{}) error {
	s.LockStorage()
	s.InterfaceMap[key] = element
	err := s.storeSession()
	s.UnlockStorage()
	return err
}

//Pulls an element from the interface in the map
func (s *SessionObj) QueryMap(key string) (interface{}, error) {
	var err error
	s.LockStorage()
	element, ok := s.InterfaceMap[key]
	if !ok {
		err = ErrQuery
		element = nil
	}
	s.UnlockStorage()
	return element, err
}

func (s *SessionObj) DeleteMap(key string) error {
	s.LockStorage()
	delete(s.InterfaceMap, key)
	err := s.storeSession()
	s.UnlockStorage()
	return err
}

func (s *SessionObj) GetSessionData() ([]byte, error) {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.getSessionData()
}

func (s *SessionObj) GetKeyStore() *keyStore.KeyStore {
	return s.KeyMaps
}

func (s *SessionObj) GetRekeyManager() *keyStore.RekeyManager {
	return s.RekeyManager
}

func (s *SessionObj) GetSwitchboard() *switchboard.Switchboard {
	return s.listeners
}

func (s *SessionObj) GetQuitChan() chan bool {
	return s.quitReceptionRunner
}

func (s *SessionObj) getSessionData() ([]byte, error) {
	var session bytes.Buffer

	enc := gob.NewEncoder(&session)

	err := enc.Encode(s)

	if err != nil {
		err = errors.New(fmt.Sprintf("StoreSession: Could not encode user"+
			" session: %s", err.Error()))
		return nil, err
	}
	return session.Bytes(), nil
}

// Locking a mutex that belongs to the session object makes the locking
// independent of the implementation of the storage, which is probably good.
func (s *SessionObj) LockStorage() {
	s.lock.Lock()
}

func (s *SessionObj) UnlockStorage() {
	s.lock.Unlock()
}

func clearCyclicInt(c *cyclic.Int) {
	c.Reset()
	//c.Set(cyclic.NewMaxInt())
	//c.SetInt64(0)
}

// FIXME Shouldn't we just be putting pseudorandom bytes in to obscure the mem?
func burntString(length int) string {

	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}
