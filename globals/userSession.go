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
	"math"
	"math/rand"
	"time"
)

// Globally instantiated UserSession
var Session UserSession

// Interface for User Session operations
type UserSession interface {
	GetCurrentUser() (currentUser *User)
	GetNodeAddress() string
	GetKeys() []NodeKeys
	GetPrivateKey() *cyclic.Int
	PushFifo(*Message) error
	PopFifo() (*Message, error)
	StoreSession() error
	Immolate() error
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

var FifoEmptyErr error = errors.New("PopFifo: Fifo Empty")

// Creates a new UserSession interface for registration
func NewUserSession(u *User, nodeAddr string, nk []NodeKeys) UserSession {

	// With an underlying Session data structure
	return UserSession(&sessionObj{
		CurrentUser: u,
		NodeAddress: nodeAddr,
		fifo:        nil,
		Keys:        nk,
		pollTerm:    nil,
		PrivateKey:  cyclic.NewMaxInt()})
}

func LoadSession(UID uint64, pollTerm ThreadTerminator) error {
	if LocalStorage == nil {
		err := errors.New("StoreSession: Local Storage not avalible")
		return err
	}

	rand.Seed(time.Now().UnixNano())

	sessionGob := LocalStorage.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Write(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := sessionObj{}

	err := dec.Decode(&session)

	if (err != nil && err != io.EOF) || (session.CurrentUser == nil) {
		err = errors.New(fmt.Sprintf("LoadSession: unable to load session: %s", err.Error()))
		return err
	} else if err != nil {
		err = errors.New(fmt.Sprintf("LoadSession: unknown error: %s",
			err.Error()))
		return err
	}

	if session.CurrentUser.UserID != UID {
		err = errors.New(fmt.Sprintf(
			"LoadSession: loaded incorrect + +"+
				"user; Expected: %v; Received: %v", UID,
			session.CurrentUser.UserID))
		return err
	}

	session.fifo = make(chan *Message, 100)

	session.pollTerm = pollTerm

	Session = &session

	return nil
}

// Struct holding relevant session data
type sessionObj struct {
	// Currently authenticated user
	CurrentUser *User

	//fifo buffer
	fifo chan *Message

	// Node address that the user will send messages to
	NodeAddress string

	// Used to kill the polling reception thread
	pollTerm ThreadTerminator

	Keys       []NodeKeys
	PrivateKey *cyclic.Int
}

func (s *sessionObj) GetKeys() []NodeKeys {
	return s.Keys
}

func (s *sessionObj) GetPrivateKey() *cyclic.Int {
	return s.PrivateKey
}

// Return a copy of the current user
func (s *sessionObj) GetCurrentUser() (currentUser *User) {
	if s.CurrentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			UserID: s.CurrentUser.UserID,
			Nick:   s.CurrentUser.Nick,
		}
	}
	return
}

func (s *sessionObj) GetNodeAddress() string {
	return s.NodeAddress
}

func (s *sessionObj) PushFifo(msg *Message) error {

	if s.fifo == nil {
		err := errors.New("PushFifo: Cannot push an uninitialized fifo")
		return err
	}

	if s.CurrentUser == nil {
		err := errors.New("PushFifo: Cannot push a fifo for an uninitialized")
		return err
	}

	select {
	case s.fifo <- msg:
		return nil
	default:
		err := errors.New("PushFifo: fifo full")
		return err
	}
}

func (s *sessionObj) PopFifo() (*Message, error) {

	if s.fifo == nil {
		err := errors.New("PopFifo: Cannot pop an uninitialized fifo")
		return nil, err
	}

	if s.CurrentUser == nil {
		err := errors.New("PopFifo: Cannot pop an fifo on an uninitialized" +
			" user")
		return nil, err
	}

	var msg *Message

	select {
	case msg = <-s.fifo:
		return msg, nil
	default:
		err := errors.New("PopFifo: Fifo Empty")
		return nil, err
	}

}

func (s *sessionObj) StoreSession() error {

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

// Scrubs all cryptographic data from ram and logs out
// the ram overwriting can be improved
func (s *sessionObj) Immolate() error {
	if s == nil {
		err := errors.New("immolate: Cannot immolate when" +
			" you are not alive")
		return err
	}

	//Kill Polling Reception
	if s.pollTerm != nil {

		s.pollTerm.BlockingTerminate(60000)
		//Clear message fifo

		q := false
		for !q {
			select {
			case m := <-s.fifo:
				clearCyclicInt(m.payload)
				clearCyclicInt(m.senderID)
				clearCyclicInt(m.recipientInitVect)
				clearCyclicInt(m.recipientID)
				clearCyclicInt(m.payloadInitVect)
			default:
				q = true
			}
		}

		//close the message fifo
		close(s.fifo)
	}

	// clear data stored in session
	s.CurrentUser.UserID = math.MaxUint64
	s.CurrentUser.UserID = 0
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = ""

	s.NodeAddress = burntString(len(s.NodeAddress))
	s.NodeAddress = burntString(len(s.NodeAddress))
	s.NodeAddress = ""

	clearCyclicInt(s.PrivateKey)

	for i := 0; i < len(s.Keys); i++ {
		clearCyclicInt(s.Keys[i].PublicKey)
		clearRatchetKeys(&s.Keys[i].TransmissionKeys)
		clearRatchetKeys(&s.Keys[i].ReceptionKeys)
	}

	Session = nil

	return nil
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
