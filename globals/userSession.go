////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally instantiated UserSession
var Session UserSession

// Interface for User Session operations
type UserSession interface {
	Login(id uint64, addr string) (isValidUser bool)
	GetCurrentUser() (currentUser *User)
	GetNodeAddress() string
	GetKeys() []NodeKeys
	GetPrivateKey() *cyclic.Int
	PushFifo(*Message)
	PopFifo() *Message
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

// Creates a new UserSession interface
func newUserSession(numNodes int) UserSession {
	keySlc := make([]NodeKeys, numNodes)

	for i := 0; i < numNodes; i++ {
		keySlc[i] = NodeKeys{PublicKey: cyclic.NewMaxInt(),
			TransmissionKeys: RatchetKey{
				Base: cyclic.NewIntFromString(
					"c1248f42f8127999e07c657896a26b56fd9a499c6199e1265053132451128f52", 16),
				Recursive: cyclic.NewIntFromString(
					"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251", 16)},
			ReceptionKeys: RatchetKey{
				Base: cyclic.NewIntFromString(
					"83120e7bfaba497f8e2c95457a28006f73ff4ec75d3ad91d27bf7ce8f04e772c", 16),
				Recursive: cyclic.NewIntFromString(
					"979e574166ef0cd06d34e3260fe09512b69af6a414cf481770600d9c7447837b", 16)},
			ReceiptKeys: RatchetKey{
				Base: cyclic.NewIntFromString(
					"de9a521b7d86d7706e9e0e23b072348e268b1afd5c987a295026e2baa808b78e", 16),
				Recursive: cyclic.NewIntFromString(
					"9b455586c58c77c0ff59520bfd7771d3f8dc4bddb63707cd7930a711f155ab8c", 16)},
			ReturnKeys: RatchetKey{
				Base: cyclic.NewIntFromString(
					"fa7fe4aea8c9f57d462b2902fb6ef7235be7d5b62ceb10fee3a2852ad799bbbc", 16),
				Recursive: cyclic.NewIntFromString(
					"2af0a99575b36d39acc1e97df58f8655438f716134a693ffea03e2ce519870ce", 16)}}
	}

	// With an underlying Session data structure
	return UserSession(&sessionObj{
		currentUser: nil,
		fifo:        make(chan *Message, 100),
		keys:        keySlc,
		privateKey:  cyclic.NewMaxInt()})
}

// Struct holding relevant session data
type sessionObj struct {
	// Currently authenticated user
	currentUser *User

	//Fifo buffer
	fifo chan *Message

	// Node address that the user will send messages to
	nodeAddress string

	keys       []NodeKeys
	privateKey *cyclic.Int

	grp cyclic.Group
}

func (s *sessionObj) GetKeys() []NodeKeys {
	return s.keys
}

func (s *sessionObj) GetPrivateKey() *cyclic.Int {
	return s.privateKey
}

func InitSession(numNodes int) {
	Session = newUserSession(numNodes)
}

// Set CurrentUser to the user corresponding to the given id
// if it exists. Return a bool for whether the given id exists
func (s *sessionObj) Login(id uint64, addr string) (isValidUser bool) {
	user, userExists := Users.GetUser(id)
	// User must exist and no User can be previously logged in
	if isValidUser = userExists && s.GetCurrentUser() == nil; isValidUser {
		s.currentUser = user
	}

	s.nodeAddress = addr

	initCrypto()

	return
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

func (s *sessionObj) PushFifo(msg *Message) {

	if s.currentUser == nil {
		return
	}

	select {
	case s.fifo <- msg:
		return
	default:
		return
	}
}

func (s *sessionObj) PopFifo() *Message {

	if s.currentUser == nil {
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
