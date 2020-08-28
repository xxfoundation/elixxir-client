////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
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
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"sync"
	"time"
)

// Errors
var ErrQuery = errors.New("element not in map")

// Interface for User Session operations
type Session interface {
	StoreSession() error
	Immolate() error
	GetKeyStore() *keyStore.KeyStore
	GetRekeyManager() *keyStore.RekeyManager
	LockStorage()
	UnlockStorage()
	GetSessionData() ([]byte, error)
	StorageIsEmpty() bool
	GetSessionLocation() uint8
	LoadEncryptedSession(store globals.Storage) ([]byte, error)
	SetE2EGrp(g *cyclic.Group)
	SetUser(u *id.ID)
}

type NodeKeys struct {
	TransmissionKey *cyclic.Int
	ReceptionKey    *cyclic.Int
}

// Creates a new Session interface for registration
func NewSession(store globals.Storage,
	password string) Session {
	// With an underlying Session data structure
	return Session(&SessionObj{
		KeyMaps:         keyStore.NewStore(),
		RekeyManager:    keyStore.NewRekeyManager(),
		store:           store,
		password:        password,
		storageLocation: globals.LocationA,
	})
}

//LoadSession loads the encrypted session from the storage location and processes it
// Returns a session object on success
func LoadSession(store globals.Storage, password string) (Session, error) {
	if store == nil {
		err := errors.New("LoadSession: Local Storage not available")
		return nil, err
	}

	wrappedSession, loadLocation, err := processSession(store, password)
	if err != nil {
		return nil, err
	}

	for wrappedSession.Version != SessionVersion {
		switch wrappedSession.Version {
		case 1:
			globals.Log.INFO.Println("Converting session file from V1 to V2")
			wrappedSession, err = ConvertSessionV1toV2(wrappedSession)
		default:
		}
		if err != nil {
			return nil, err
		}
	}

	//extract the session from the wrapper
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
		session.CurrentUser)
	// Create switchboard
	//session.listeners = switchboard.New()
	// Create quit channel for reception runner
	//session.quitReceptionRunner = make(chan struct{})

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

//processSessionWrapper acts as a helper function for processSession
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
// When adding to this structure, ALWAYS ALWAYS
// consider if you want the data to be in the session file
type SessionObj struct {
	// E2E KeyStore
	KeyMaps *keyStore.KeyStore

	// do not touch until removing session, neeeded for keystores
	E2EGrp      *cyclic.Group
	CurrentUser *id.ID

	// Rekey Manager
	RekeyManager *keyStore.RekeyManager

	// Non exported fields (not GOB encoded/decoded)
	// Local pointer to storage of this session
	store globals.Storage

	lock sync.Mutex

	// The password used to encrypt this session when saved
	password string

	storageLocation uint8
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

func (s *SessionObj) SetE2EGrp(g *cyclic.Group) {
	s.E2EGrp = g
}

func (s *SessionObj) SetUser(u *id.ID) {
	s.CurrentUser = u
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
	Id id.ID
	Pk []byte
}

func (s *SessionObj) StorageIsEmpty() bool {
	s.LockStorage()
	defer s.UnlockStorage()
	return s.store.IsEmpty()
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
	return nil //s.listeners
}

func (s *SessionObj) GetQuitChan() chan struct{} {
	return nil //s.quitReceptionRunner
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

func (s *SessionObj) GetSessionLocation() uint8 {
	if s.storageLocation == globals.LocationA {
		return globals.LocationA
	} else if s.storageLocation == globals.LocationB {
		return globals.LocationB
	}
	return globals.NoSave
}
