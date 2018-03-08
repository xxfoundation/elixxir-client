////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"bytes"
	"encoding/gob"
	jww "github.com/spf13/jwalterweatherman"
	"math"
	"time"
	"math/rand"
	"io"
)

// Globally instantiated UserSession
var Session UserSession

// Interface for User Session operations
type UserSession interface {
	GetCurrentUser() (currentUser *User)
	GetNodeAddress() string
	GetKeys() []NodeKeys
	GetPrivateKey() *cyclic.Int
	PushFifo(*Message) (bool)
	PopFifo() *Message
	StoreSession() bool
	Immolate() bool
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

func LoadSession(UID uint64, pollTerm ThreadTerminator)(bool){
	if LocalStorage == nil {
		jww.ERROR.Println("StoreSession: Local Storage not avalible")
		return false
	}

	rand.Seed(time.Now().UnixNano())

	sessionGob := LocalStorage.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Write(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	session := sessionObj{}

	err := dec.Decode(&session)


	if err!=nil && err!=io.EOF {
		jww.ERROR.Printf("LoadSession: unable to load session: %s", err.Error())
		return false
	}

	if session.CurrentUser == nil {
		jww.ERROR.Println("LoadSession: failed to load user %v", session)

		return false
	}

	if session.CurrentUser.UID!=UID{
		jww.ERROR.Printf("LoadSession: loaded incorrect user; Expected: %v; Received: %v", UID, session.CurrentUser.UID)
		return false
	}


	session.fifo = make(chan *Message, 100)

	session.pollTerm = pollTerm

	Session = &session

	return true
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
			UID:   s.CurrentUser.UID,
			Nick: s.CurrentUser.Nick,
		}
	}
	return
}

func (s *sessionObj) GetNodeAddress() string {
	return s.NodeAddress
}

func (s *sessionObj) PushFifo(msg *Message)(bool) {

	if s.fifo == nil {
		jww.ERROR.Println("PushFifo: Cannot push an uninitialized fifo")
		return false
	}

	if s.CurrentUser == nil {
		jww.ERROR.Println("PushFifo: Cannot push a fifo for an uninitialized" +
			" user")
		return false
	}

	select {
	case s.fifo <- msg:
		return true
	default:
		jww.ERROR.Println("PushFifo: fifo full")
		return false
	}
}

func (s *sessionObj) PopFifo() *Message {

	if s.fifo == nil {
		jww.ERROR.Println("PopFifo: Cannot pop an uninitialized fifo")
		return nil
	}


	if s.CurrentUser == nil {
		jww.ERROR.Println("PopFifo: Cannot pop an fifo on an uninitialized" +
			" user")
		return nil
	}

	var msg *Message

	select {
	case msg = <-s.fifo:
		return msg
	default:
		return nil
	}

}


func (s *sessionObj) StoreSession()(bool){

	if LocalStorage == nil {
		jww.ERROR.Println("StoreSession: Local Storage not avalible")
		return false
	}

	var session bytes.Buffer

	enc := gob.NewEncoder(&session)

	err := enc.Encode(s)


	if err!=nil{
		jww.ERROR.Println("StoreSession: Could not encode user" +
			" session: %s", err.Error())
		return false
	}


	LocalStorage, err = LocalStorage.Save(session.Bytes())

	if err!= nil{
		jww.ERROR.Println("StoreSession: Could not save the encoded user" +
			" session")
		return false
	}

	return true

}



// Scrubs all cryptographic data from ram and logs out
// the ram overwriting can be improved
func (s *sessionObj) Immolate()(bool) {
	if s == nil {
		jww.ERROR.Println("CryptographicallyImmolate: Cannot immolate when" +
			" you are not alive")
		return false
	}

	//Kill Polling Reception
	if s.pollTerm != nil{

		s.pollTerm.BlockingTerminate(1000)
		//Clear message fifo
		for {
			select {
			case m := <-s.fifo:
				clearCyclicInt(m.payload)
				clearCyclicInt(m.senderID)
				clearCyclicInt(m.recipientInitVect)
				clearCyclicInt(m.recipientID)
				clearCyclicInt(m.payloadInitVect)
			default:
				break
			}
		}

		//close the message fifo
		close(s.fifo)
	}




	// clear data stored in session
	s.CurrentUser.UID = math.MaxUint64
	s.CurrentUser.UID = 0
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = burntString(len(s.CurrentUser.Nick))
	s.CurrentUser.Nick = ""

	s.NodeAddress = burntString(len(s.NodeAddress))
	s.NodeAddress = burntString(len(s.NodeAddress))
	s.NodeAddress = ""

	clearCyclicInt(s.PrivateKey)

	for i:=0;i<len(s.Keys);i++{
		clearCyclicInt(s.Keys[i].PublicKey)
		clearRatchetKeys(&s.Keys[i].TransmissionKeys)
		clearRatchetKeys(&s.Keys[i].ReceptionKeys)
	}

	Session = nil

	return true
}


func clearCyclicInt(c *cyclic.Int){
	c.Set(cyclic.NewMaxInt())
	c.SetInt64(0)
}

func clearRatchetKeys(r *RatchetKey){
	clearCyclicInt(r.Base)
	clearCyclicInt(r.Recursive)
}

func burntString(length int)string{

	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}
