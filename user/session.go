////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
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
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Errors
var ErrQuery = errors.New("element not in map")

// Interface for User Session operations
type Session interface {
	GetCurrentUser() (currentUser *User)
	GetNodeKeys(topology *connect.Circuit) []NodeKeys
	PushNodeKey(id *id.Node, key NodeKeys)
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
	AppendGarbledMessage(messages ...*format.Message)
	PopGarbledMessages() []*format.Message
	GetSalt() []byte
	SetRegState(rs uint32) error
	GetRegState() uint32
	ChangeUsername(string) error
	StorageIsEmpty() bool
	GetContactByValue(string) (*id.User, []byte)
	StoreContactByValue(string, *id.User, []byte)
	DeleteContact(*id.User) (string, error)
	GetSessionLocation() uint8
	LoadEncryptedSession(store globals.Storage) ([]byte, error)
	RegisterPermissioningSignature(sig []byte) error
}

type NodeKeys struct {
	TransmissionKey *cyclic.Int
	ReceptionKey    *cyclic.Int
}

// Creates a new Session interface for registration
func NewSession(store globals.Storage,
	u *User,
	publicKeyRSA *rsa.PublicKey,
	privateKeyRSA *rsa.PrivateKey,
	cmixPublicKeyDH *cyclic.Int,
	cmixPrivateKeyDH *cyclic.Int,
	e2ePublicKeyDH *cyclic.Int,
	e2ePrivateKeyDH *cyclic.Int,
	salt []byte,
	cmixGrp, e2eGrp *cyclic.Group,
	password string) Session {
	regState := uint32(KeyGenComplete)
	// With an underlying Session data structure
	return Session(&SessionObj{
		NodeKeys:         make(map[id.Node]NodeKeys),
		CurrentUser:      u,
		RSAPublicKey:     publicKeyRSA,
		RSAPrivateKey:    privateKeyRSA,
		CMIXDHPublicKey:  cmixPublicKeyDH,
		CMIXDHPrivateKey: cmixPrivateKeyDH,
		E2EDHPublicKey:   e2ePublicKeyDH,
		E2EDHPrivateKey:  e2ePrivateKeyDH,
		CmixGrp:          cmixGrp,
		E2EGrp:           e2eGrp,
		InterfaceMap:     make(map[string]interface{}),
		KeyMaps:          keyStore.NewStore(),
		RekeyManager:     keyStore.NewRekeyManager(),
		store:                  store,
		listeners:              switchboard.NewSwitchboard(),
		quitReceptionRunner:    make(chan struct{}),
		password:               password,
		Salt:                   salt,
		RegState:               &regState,
		storageLocation:        globals.LocationA,
		ContactsByValue:        make(map[string]SearchedUserRecord),
	})
}

func LoadSession(store globals.Storage, password string) (Session, error) {
	if store == nil {
		err := errors.New("LoadSession: Local Storage not available")
		return nil, err
	}

	wrappedSession, loadLocation, err := processSession(store, password)
	if err != nil {
		return nil, err
	}

	//extract teh session from the wrapper
	var sessionBytes bytes.Buffer

	sessionBytes.Write(wrappedSession.Session)
	dec := gob.NewDecoder(&sessionBytes)

	session := SessionObj{}

	err = dec.Decode(&session)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to decode session")
	}

	session.storageLocation = loadLocation

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

//processSession: gets the loadLocation and decrypted wrappedSession
func processSession(store globals.Storage, password string) (*SessionStorageWrapper, uint8, error) {
	var wrappedSession *SessionStorageWrapper
	loadLocation := globals.NoSave
	//load sessions
	wrappedSessionA, errA := processSessionWrapper(store.LoadA(), password)
	wrappedSessionB, errB := processSessionWrapper(store.LoadB(), password)

	//figure out which session to use of the two locations
	if errA != nil && errB != nil {
		return nil, globals.NoSave, errors.Errorf("Loading both sessions errored: \n "+
			"SESSION A ERR: %s \n SESSION B ERR: %s", errA, errB)
	} else if errA == nil && errB != nil {
		loadLocation = globals.LocationA
		wrappedSession = wrappedSessionA
	} else if errA != nil && errB == nil {
		loadLocation = globals.LocationB
		wrappedSession = wrappedSessionB
	} else {
		if wrappedSessionA.Timestamp.After(wrappedSessionB.Timestamp) {
			loadLocation = globals.LocationA
			wrappedSession = wrappedSessionA
		} else {
			loadLocation = globals.LocationB
			wrappedSession = wrappedSessionB
		}
	}
	return wrappedSession, loadLocation, nil

}

func processSessionWrapper(sessionGob []byte, password string) (*SessionStorageWrapper, error) {

	if sessionGob == nil || len(sessionGob) < 12 {
		return nil, errors.New("No session file passed")
	}

	decryptedSessionGob, err := decrypt(sessionGob, password)

	if err != nil {
		return nil, errors.Wrap(err, "Could not decode the "+
			"session wrapper")
	}

	var sessionBytes bytes.Buffer

	sessionBytes.Write(decryptedSessionGob)
	dec := gob.NewDecoder(&sessionBytes)

	wrappedSession := SessionStorageWrapper{}

	err = dec.Decode(&wrappedSession)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to decode session wrapper")
	}

	return &wrappedSession, nil
}

// Struct holding relevant session data
type SessionObj struct {
	// Currently authenticated user
	CurrentUser *User

	NodeKeys         map[id.Node]NodeKeys
	RSAPrivateKey    *rsa.PrivateKey
	RSAPublicKey     *rsa.PublicKey
	CMIXDHPrivateKey *cyclic.Int
	CMIXDHPublicKey  *cyclic.Int
	E2EDHPrivateKey  *cyclic.Int
	E2EDHPublicKey   *cyclic.Int
	CmixGrp          *cyclic.Group
	E2EGrp           *cyclic.Group
	Salt             []byte

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

	// Buffer of messages that cannot be decrypted
	garbledMessages []*format.Message

	RegState *uint32

	storageLocation uint8

	ContactsByValue map[string]SearchedUserRecord
}

//WriteToSession: Writes to the location where session is being stored the arbitrary replacement string
// The replacement string is meant to be the output of a loadEncryptedSession
func WriteToSession(replacement []byte, store globals.Storage) error {
	//Write to both
	err := store.SaveA(replacement)
	if err != nil {
		return errors.Errorf("Failed to save to session A: %v", err)
	}
	err = store.SaveB(replacement)
	if err != nil {
		return errors.Errorf("Failed to save to session B: %v", err)
	}

	return nil
}


//LoadEncryptedSession: gets the encrypted session file from storage
// Returns it as a base64 encoded string
func (s *SessionObj) LoadEncryptedSession(store globals.Storage) ([]byte, error) {
	sessionData, _, err := processSession(store, s.password)
	if err != nil {
		return make([]byte, 0), err
	}
	encryptedSession := encrypt(sessionData.Session, s.password)
	return encryptedSession, nil
}

type SearchedUserRecord struct {
	Id id.User
	Pk []byte
}

func (s *SessionObj) GetLastMessageID() string {
	s.LockStorage()
	defer s.UnlockStorage()

	return s.LastMessageID
}

func (s *SessionObj) StorageIsEmpty() bool {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.store.IsEmpty()
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
	for node := range s.NodeKeys {
		nodes[node] = 1
	}
	return nodes
}

func (s *SessionObj) GetSalt() []byte {
	s.LockStorage()
	defer s.UnlockStorage()
	salt := make([]byte, len(s.Salt))
	copy(salt, s.Salt)
	return salt
}

func (s *SessionObj) GetNodeKeys(topology *connect.Circuit) []NodeKeys {
	s.LockStorage()
	defer s.UnlockStorage()

	keys := make([]NodeKeys, topology.Len())

	for i := 0; i < topology.Len(); i++ {
		keys[i] = s.NodeKeys[*topology.GetNodeAtIndex(i)]
	}

	return keys
}

func (s *SessionObj) PushNodeKey(id *id.Node, key NodeKeys) {
	s.LockStorage()
	defer s.UnlockStorage()

	s.NodeKeys[*id] = key
}

func (s *SessionObj) RegisterPermissioningSignature(sig []byte) error {
	s.LockStorage()
	defer s.UnlockStorage()

	err := s.SetRegState(PermissioningComplete)
	if err != nil {
		return errors.Wrap(err, "Could not store permissioning signature")
	}

	s.regValidationSignature = sig

	return nil
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
			User:     s.CurrentUser.User,
			Username: s.CurrentUser.Username,
		}
	}
	return currentUser
}

func (s *SessionObj) GetRegState() uint32 {
	return atomic.LoadUint32(s.RegState)
}

func (s *SessionObj) SetRegState(rs uint32) error {
	prevRs := rs - 1
	b := atomic.CompareAndSwapUint32(s.RegState, prevRs, rs)
	if !b {
		return errors.New("Could not increment registration state")
	}
	return nil
}

func (s *SessionObj) ChangeUsername(username string) error {
	b := s.GetRegState()
	if b != PermissioningComplete {
		return errors.New("Can only change username during " +
			"PermissioningComplete registration state")
	}
	s.CurrentUser.Username = username
	return nil
}

type SessionStorageWrapper struct {
	Version   uint32
	Timestamp time.Time
	Session   []byte
}

func (s *SessionObj) storeSession() error {

	if s.store == nil {
		err := errors.New("StoreSession: Local Storage not available")
		return err
	}

	sessionData, err := s.getSessionData()

	encryptedSession := encrypt(sessionData, s.password)
	if s.storageLocation == globals.LocationA {
		err = s.store.SaveB(encryptedSession)
		if err != nil {
			err = errors.New(fmt.Sprintf("StoreSession: Could not save the encoded user"+
				" session in location B: %s", err.Error()))
		} else {
			s.storageLocation = globals.LocationB
		}
	} else if s.storageLocation == globals.LocationB {
		err = s.store.SaveA(encryptedSession)
		if err != nil {
			err = errors.New(fmt.Sprintf("StoreSession: Could not save the encoded user"+
				" session in location A: %s", err.Error()))
		} else {
			s.storageLocation = globals.LocationA
		}
	} else {
		err = errors.New("Could not store because no location is " +
			"selected")
	}

	return err

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

	globals.Log.WARN.Println("Immolate not implemented, did nothing")

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
	var sessionBuffer bytes.Buffer

	enc := gob.NewEncoder(&sessionBuffer)

	err := enc.Encode(s)

	if err != nil {
		err = errors.New(fmt.Sprintf("StoreSession: Could not encode user"+
			" session: %s", err.Error()))
		return nil, err
	}

	sw := SessionStorageWrapper{
		Version:   SessionVersion,
		Session:   sessionBuffer.Bytes(),
		Timestamp: time.Now(),
	}

	var wrapperBuffer bytes.Buffer

	enc = gob.NewEncoder(&wrapperBuffer)

	err = enc.Encode(&sw)

	if err != nil {
		err = errors.New(fmt.Sprintf("StoreSession: Could not encode user"+
			" session wrapper: %s", err.Error()))
		return nil, err
	}

	return wrapperBuffer.Bytes(), nil
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
		return nil, errors.Wrap(err, "Cannot decrypt with password!")
	}
	return plaintext, nil
}

// AppendGarbledMessage appends a message or messages to the garbled message
// buffer.
// FIXME: improve performance of adding items to the buffer
func (s *SessionObj) AppendGarbledMessage(messages ...*format.Message) {
	s.garbledMessages = append(s.garbledMessages, messages...)
}

// PopGarbledMessages returns the content of the garbled message buffer and
// deletes its contents.
func (s *SessionObj) PopGarbledMessages() []*format.Message {
	tempBuffer := s.garbledMessages
	s.garbledMessages = []*format.Message{}
	return tempBuffer
}

func (s *SessionObj) GetContactByValue(v string) (*id.User, []byte) {
	s.LockStorage()
	defer s.UnlockStorage()
	u, ok := s.ContactsByValue[v]
	if !ok {
		return nil, nil
	}
	return &(u.Id), u.Pk
}

func (s *SessionObj) StoreContactByValue(v string, uid *id.User, pk []byte) {
	s.LockStorage()
	defer s.UnlockStorage()
	u, ok := s.ContactsByValue[v]
	if ok {
		globals.Log.WARN.Printf("Attempted to store over extant "+
			"user value: %s; before: %v, new: %v", v, u.Id, *uid)
	} else {
		s.ContactsByValue[v] = SearchedUserRecord{
			Id: *uid,
			Pk: pk,
		}
	}
}

func (s *SessionObj) DeleteContact(uid *id.User) (string, error) {
	s.LockStorage()
	defer s.UnlockStorage()

	for v, u := range s.ContactsByValue {
		if u.Id.Cmp(uid) {
			delete(s.ContactsByValue, v)
			_, ok := s.ContactsByValue[v]
			if ok {
				return "", errors.Errorf("Failed to delete user: %+v", u)
			} else {
				return v, nil
			}
		}
	}

	return "", errors.Errorf("No user found in usermap with userid: %s",
		uid)

}

func (s *SessionObj) GetSessionLocation() uint8 {
	if s.storageLocation == globals.LocationA {
		return globals.LocationA
	} else if s.storageLocation == globals.LocationB {
		return globals.LocationB
	}
	return globals.NoSave
}
