////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"gitlab.com/privategrity/crypto/cyclic"
	"io"
	"math/rand"
	"time"
	"math"
)

// Globally instantiated UserSession
var Session UserSession

// Interface for User Session operations
type UserSession interface {
	GetCurrentUser() (currentUser *User)
	GetGWAddress() string
	SetGWAddress(addr string)
	GetKeys() []NodeKeys
	GetPrivateKey() *cyclic.Int
	GetPublicKey() *cyclic.Int
	StoreSession() error
	Immolate() error
	UpsertMap(key string, element interface{}) error
	QueryMap(key string) (interface{}, error)
}

type NodeKeys struct {
	PublicKey        *cyclic.Int
	TransmissionKeys RatchetKey
	ReceptionKeys    RatchetKey
	ReceiptKeys      RatchetKey
	ReturnKeys       RatchetKey
}

type RatchetKey struct {
	Base      *cyclic.Int
	Recursive *cyclic.Int
}

// Creates a new UserSession interface for registration
func NewUserSession(u *User, GatewayAddr string, nk []NodeKeys) UserSession {

	// With an underlying Session data structure
	return UserSession(&SessionObj{
		CurrentUser:  u,
		GWAddress:    GatewayAddr, // FIXME: don't store this here
		Keys:         nk,
		PrivateKey:   cyclic.NewMaxInt(),
		InterfaceMap: make(map[string]interface{}),
	})

}

func LoadSession(UID UserID) error {
	if LocalStorage == nil {
		err := errors.New("StoreSession: Local Storage not avalible")
		return err
	}

	rand.Seed(time.Now().UnixNano())

	sessionGob := LocalStorage.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Write(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := SessionObj{}

	err := dec.Decode(&session)

	if (err != nil && err != io.EOF) || (session.CurrentUser == nil) {
		err = errors.New(fmt.Sprintf("LoadSession: unable to load session: %s", err.Error()))
		return err
	}

	if session.CurrentUser.UserID != UID {
		err = errors.New(fmt.Sprintf(
			"LoadSession: loaded incorrect "+
				"user; Expected: %v; Received: %v",
			session.CurrentUser.UserID, UID))
		return err
	}

	Session = &session

	return nil
}

// Struct holding relevant session data
type SessionObj struct {
	// Currently authenticated user
	CurrentUser *User

	// Gateway address to the cMix network
	GWAddress string

	Keys       []NodeKeys
	PrivateKey *cyclic.Int

	//Interface map for random data storage
	InterfaceMap map[string]interface{}
}

func (s *SessionObj) GetKeys() []NodeKeys {
	return s.Keys
}

func (s *SessionObj) GetPrivateKey() *cyclic.Int {
	return s.PrivateKey
}

func (s *SessionObj) GetPublicKey() *cyclic.Int {
	return cyclic.NewMaxInt()
}

// Return a copy of the current user
func (s *SessionObj) GetCurrentUser() (currentUser *User) {
	if s.CurrentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			UserID: s.CurrentUser.UserID,
			Nick:   s.CurrentUser.Nick,
		}
	}
	return
}

func (s *SessionObj) GetGWAddress() string {
	return s.GWAddress
}

func (s *SessionObj) SetGWAddress(addr string) {
	s.GWAddress = addr
}

func (s *SessionObj) StoreSession() error {

	if LocalStorage == nil {
		err := errors.New("StoreSession: Local Storage not available")
		return err
	}

	var session bytes.Buffer

	enc := gob.NewEncoder(&session)

	err := enc.Encode(s)

	if err != nil {
		err = errors.New(fmt.Sprintf("StoreSession: Could not encode user"+
			" session: %s", err.Error()))
		return err
	}

	err = LocalStorage.Save(session.Bytes())

	if err != nil {

		err = errors.New(fmt.Sprintf("StoreSession: Could not save the encoded user"+
			" session: %s", err.Error()))
		return err
	}

	return nil

}

// Immolate scrubs all cryptographic data from ram and logs out
// the ram overwriting can be improved
func (s *SessionObj) Immolate() error {
	if s == nil {
		err := errors.New("immolate: Cannot immolate that which has no life")
		return err
	}

	// clear data stored in session
	s.CurrentUser.UserID = math.MaxUint64
	s.CurrentUser.UserID = 0
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = ""

	s.GWAddress = burntString(len(s.GWAddress))
	s.GWAddress = burntString(len(s.GWAddress))
	s.GWAddress = ""

	clearCyclicInt(s.PrivateKey)

	for i := 0; i < len(s.Keys); i++ {
		clearCyclicInt(s.Keys[i].PublicKey)
		clearRatchetKeys(&s.Keys[i].TransmissionKeys)
		clearRatchetKeys(&s.Keys[i].ReceptionKeys)
	}

	Session = nil

	return nil
}

//Upserts an element into the interface map and saves the session object
func (s *SessionObj) UpsertMap(key string, element interface{}) error {
	s.InterfaceMap[key] = element
	err := s.StoreSession()
	return err
}

//Pulls an element from the interface in the map
func (s *SessionObj) QueryMap(key string) (interface{}, error) {
	element, ok := s.InterfaceMap[key]
	if !ok {
		return nil, errors.New("element not in map")
	}
	return element, nil
}

func clearCyclicInt(c *cyclic.Int) {
	c.Set(cyclic.NewMaxInt())
	c.SetInt64(0)
}

func clearRatchetKeys(r *RatchetKey) {
	clearCyclicInt(r.Base)
	clearCyclicInt(r.Recursive)
}

func burntString(length int) string {

	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}
