////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package user

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"io"
	"sync"
)

// Errors
var ErrQuery = errors.New("element not in map")

// Interface for User Session operations
type Session interface {
	GetCurrentUser() (currentUser *User)
	GetKeys(topology *circuit.Circuit) []NodeKeys
	GetRSAPrivateKey() *rsa.PrivateKey
	GetRSAPublicKey() *rsa.PublicKey
	GetCMIXDHPrivateKey() *cyclic.Int
	GetCMIXDHPublicKey() *cyclic.Int
	GetE2EDHPrivateKey() *cyclic.Int
	GetE2EDHPublicKey() *cyclic.Int
	GetCmixGroup() *cyclic.Group
	GetE2EGroup() *cyclic.Group
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
	GetQuitChan() chan struct{}
	LockStorage()
	UnlockStorage()
	GetSessionData() ([]byte, error)
	GetRegistrationValidationSignature() []byte
	GetNodes() map[id.Node]int
}

type NodeKeys struct {
	TransmissionKey *cyclic.Int
	ReceptionKey    *cyclic.Int
}

// Creates a new Session interface for registration
func NewSession(store globals.Storage,
	u *User, nk map[id.Node]NodeKeys,
	publicKeyRSA *rsa.PublicKey,
	privateKeyRSA *rsa.PrivateKey,
	cmixPublicKeyDH *cyclic.Int,
	cmixPrivateKeyDH *cyclic.Int,
	e2ePublicKeyDH *cyclic.Int,
	e2ePrivateKeyDH *cyclic.Int,
	cmixGrp, e2eGrp *cyclic.Group,
	password string,
	regSignature []byte) Session {
	// With an underlying Session data structure
	return Session(&SessionObj{
		CurrentUser:            u,
		Keys:                   nk,
		RSAPublicKey:           publicKeyRSA,
		RSAPrivateKey:          privateKeyRSA,
		CMIXDHPublicKey:        cmixPublicKeyDH,
		CMIXDHPrivateKey:       cmixPrivateKeyDH,
		E2EDHPublicKey:         e2ePublicKeyDH,
		E2EDHPrivateKey:        e2ePrivateKeyDH,
		CmixGrp:                cmixGrp,
		E2EGrp:                 e2eGrp,
		InterfaceMap:           make(map[string]interface{}),
		KeyMaps:                keyStore.NewStore(),
		RekeyManager:           keyStore.NewRekeyManager(),
		store:                  store,
		listeners:              switchboard.NewSwitchboard(),
		quitReceptionRunner:    make(chan struct{}),
		password:               password,
		regValidationSignature: regSignature,
	})
}

func LoadSession(store globals.Storage,
	password string) (Session, error) {
	if store == nil {
		err := errors.New("LoadSession: Local Storage not avalible")
		return nil, err
	}

	sessionGob := store.Load()

	decryptedSessionGob, err := decrypt(sessionGob, password)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Could not decode the "+
			"session file %+v", err))
	}

	var sessionBytes bytes.Buffer

	sessionBytes.Write(decryptedSessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := SessionObj{}

	err = dec.Decode(&session)

	if err != nil {
		err = errors.New(fmt.Sprintf(
			"LoadSession: unable to load session: %s", err.Error()))
		return nil, err
	}

	// Reconstruct Key maps
	session.KeyMaps.ReconstructKeys(session.E2EGrp,
		session.CurrentUser.User)

	// Create switchboard
	session.listeners = switchboard.NewSwitchboard()
	// Create quit channel for reception runner
	session.quitReceptionRunner = make(chan struct{})

	// Set storage pointer
	session.store = store
	session.password = password
	return &session, nil
}

// Struct holding relevant session data
type SessionObj struct {
	// Currently authenticated user
	CurrentUser *User

	Keys             map[id.Node]NodeKeys
	RSAPrivateKey    *rsa.PrivateKey
	RSAPublicKey     *rsa.PublicKey
	CMIXDHPrivateKey *cyclic.Int
	CMIXDHPublicKey  *cyclic.Int
	E2EDHPrivateKey  *cyclic.Int
	E2EDHPublicKey   *cyclic.Int
	CmixGrp          *cyclic.Group
	E2EGrp           *cyclic.Group

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
	quitReceptionRunner chan struct{}

	lock sync.Mutex

	// The password used to encrypt this session when saved
	password string

	//The validation signature provided by permissioning
	regValidationSignature []byte
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

func (s *SessionObj) GetNodes() map[id.Node]int {
	s.LockStorage()
	defer s.UnlockStorage()
	nodes := make(map[id.Node]int, 0)
	for node, _ := range s.Keys {
		nodes[node] = 1
	}
	return nodes
}

func (s *SessionObj) GetKeys(topology *circuit.Circuit) []NodeKeys {
	s.LockStorage()
	defer s.UnlockStorage()

	keys := make([]NodeKeys, topology.Len())

	for i := 0; i < topology.Len(); i++ {
		keys[i] = s.Keys[*topology.GetNodeAtIndex(i)]
	}

	return keys
}

func (s *SessionObj) GetRSAPrivateKey() *rsa.PrivateKey {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.RSAPrivateKey
}

func (s *SessionObj) GetRSAPublicKey() *rsa.PublicKey {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.RSAPublicKey
}

func (s *SessionObj) GetCMIXDHPrivateKey() *cyclic.Int {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.CMIXDHPrivateKey
}

func (s *SessionObj) GetCMIXDHPublicKey() *cyclic.Int {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.CMIXDHPublicKey
}

func (s *SessionObj) GetE2EDHPrivateKey() *cyclic.Int {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.E2EDHPrivateKey
}

func (s *SessionObj) GetE2EDHPublicKey() *cyclic.Int {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.E2EDHPublicKey
}

func (s *SessionObj) GetCmixGroup() *cyclic.Group {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.CmixGrp
}

func (s *SessionObj) GetRegistrationValidationSignature() []byte {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.regValidationSignature
}

func (s *SessionObj) GetE2EGroup() *cyclic.Group {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.E2EGrp
}

// Return a copy of the current user
func (s *SessionObj) GetCurrentUser() (currentUser *User) {
	// This is where it deadlocks
	s.LockStorage()
	defer s.UnlockStorage()
	if s.CurrentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			User:  s.CurrentUser.User,
			Nick:  s.CurrentUser.Nick,
			Email: s.CurrentUser.Email,
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
	err = s.store.Save(encrypt(sessionData, s.password))
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

func (s *SessionObj) GetQuitChan() chan struct{} {
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
	b := make([]byte, length)

	rand.Read(b)

	return string(b)
}

// Internal crypto helper functions below

func hashPassword(password string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(password))
	return hasher.Sum(nil)
}

func initAESGCM(password string) cipher.AEAD {
	aesCipher, _ := aes.NewCipher(hashPassword(password))
	// NOTE: We use gcm as it's authenticated and simplest to set up
	aesGCM, err := cipher.NewGCM(aesCipher)
	if err != nil {
		globals.Log.FATAL.Panicf("Could not init AES GCM mode: %s",
			err.Error())
	}
	return aesGCM
}

func encrypt(data []byte, password string) []byte {
	aesGCM := initAESGCM(password)
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		globals.Log.FATAL.Panicf("Could not generate nonce: %s",
			err.Error())
	}
	ciphertext := aesGCM.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, password string) ([]byte, error) {
	aesGCM := initAESGCM(password)
	nonceLen := aesGCM.NonceSize()
	nonce, ciphertext := data[:nonceLen], data[nonceLen:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Cannot decrypt with password!"+
			" %s", err.Error()))
	}
	return plaintext, nil
}
