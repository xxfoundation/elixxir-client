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
		currentUser: u,
		nodeAddress: nodeAddr,
		fifo:        nil,
		keys:        nk,
		pollKill: 	 nil,
		privateKey:  cyclic.NewMaxInt()})
}

func LoadSession(UID uint64, pollch chan chan bool)(bool){
	if LocalStorage == nil {
		jww.ERROR.Println("StoreSession: Local Storage not avalible")
		return false
	}

	rand.Seed(time.Now().UnixNano())

	sessionGob := LocalStorage.Load()

	var sessionBytes bytes.Buffer

	sessionBytes.Read(sessionGob)

	dec := gob.NewDecoder(&sessionBytes)

	var session sessionObj

	err := dec.Decode(&session)

	if err!=nil {
		jww.ERROR.Println("LoadSession: unable to load session")
		return false
	}

	if session.currentUser.UID!=UID{
		jww.ERROR.Println("LoadSession: loaded incorrect user")
		return false
	}

	session.fifo = make(chan *Message, 100)

	session.pollKill = pollch

	Session = &session

	return true
}

// Struct holding relevant session data
type sessionObj struct {
	// Currently authenticated user
	currentUser *User

	//Fifo buffer
	fifo chan *Message

	// Node address that the user will send messages to
	nodeAddress string

	// Used to kill the polling reception thread
	pollKill chan chan bool

	keys       []NodeKeys
	privateKey *cyclic.Int
}

func (s *sessionObj) GetKeys() []NodeKeys {
	return s.keys
}

func (s *sessionObj) GetPrivateKey() *cyclic.Int {
	return s.privateKey
}

// Return a copy of the current user
func (s *sessionObj) GetCurrentUser() (currentUser *User) {
	if s.currentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			UID:   s.currentUser.UID,
			Nick: s.currentUser.Nick,
		}
	}
	return
}

func (s *sessionObj) GetNodeAddress() string {
	return s.nodeAddress
}

func (s *sessionObj) PushFifo(msg *Message)(bool) {

	if s.fifo == nil {
		jww.ERROR.Println("PushFifo: Cannot push an uninitialized fifo")
		return false
	}

	if s.currentUser == nil {
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


	if s.currentUser == nil {
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
		jww.ERROR.Println("StoreSession: Could not encode and save user" +
			" session")
		return false
	}

	LocalStorage.Save(session.Bytes())

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
	if s.pollKill != nil{
		s.blockTerminatePolling()

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
	s.currentUser.UID = math.MaxUint64
	s.currentUser.UID = 0
	s.currentUser.Nick = burntString(len(s.currentUser.Nick))
	s.currentUser.Nick = burntString(len(s.currentUser.Nick))
	s.currentUser.Nick = ""

	s.nodeAddress = burntString(len(s.nodeAddress))
	s.nodeAddress = burntString(len(s.nodeAddress))
	s.nodeAddress = ""

	clearCyclicInt(s.privateKey)

	for i:=0;i<len(s.keys);i++{
		clearCyclicInt(s.keys[i].PublicKey)
		clearRatchetKeys(&s.keys[i].TransmissionKeys)
		clearRatchetKeys(&s.keys[i].ReceptionKeys)
	}

	Session = nil

	return true
}

func (s *sessionObj) blockTerminatePolling(){
	killNotify := make(chan bool)
	s.pollKill <- killNotify
	_ = <- killNotify
	close(killNotify)
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
