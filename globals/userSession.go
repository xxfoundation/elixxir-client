package globals

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// Globally instantiated UserSession
//
var Session = newUserSession(1)

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
				Base:      cyclic.NewMaxInt(),
				Recursive: cyclic.NewMaxInt()},
			ReceptionKeys: RatchetKey{
				Base:      cyclic.NewMaxInt(),
				Recursive: cyclic.NewMaxInt()},
			ReceiptKeys: RatchetKey{
				Base:      cyclic.NewMaxInt(),
				Recursive: cyclic.NewMaxInt()},
			ReturnKeys: RatchetKey{
				Base:      cyclic.NewMaxInt(),
				Recursive: cyclic.NewMaxInt()}}
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
}

func (s *sessionObj) GetKeys() []NodeKeys {
	return s.keys
}

func (s *sessionObj) GetPrivateKey() *cyclic.Int {
	return s.privateKey
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
	return
}

// Return a copy of the current user
func (s *sessionObj) GetCurrentUser() (currentUser *User) {
	if s.currentUser != nil {
		// Explicit deep copy
		currentUser = &User{
			Id:   s.currentUser.Id,
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
