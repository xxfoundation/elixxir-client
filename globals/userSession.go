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
	GetGroup() *cyclic.Group
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

// Set CurrentUser to the user corresponding to the given id
// if it exists. Return a bool for whether the given id exists
func (s *sessionObj) Login(id uint64, addr string) (isValidUser bool) {
	user, userExists := Users.GetUser(id)
	// User must exist and no User can be previously logged in
	if isValidUser = userExists && s.GetCurrentUser() == nil; isValidUser {
		s.currentUser = user
	}

	s.nodeAddress = addr

	// TODO: don't hard code the cyclic group
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
		"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
		"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
		"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
		"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
		"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
		"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
		"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
		"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
		"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
		"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
		"FFFFFFFFFFFFFFFF"
	rng := cyclic.NewRandom(cyclic.NewInt(0), cyclic.NewInt(1000))
	s.grp = cyclic.NewGroup(cyclic.NewIntFromString(primeString, 16),
		cyclic.NewInt(5), cyclic.NewInt(4), rng)

	return
}

func (s *sessionObj) GetGroup() *cyclic.Group {
	return &s.grp
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
