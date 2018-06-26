////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/bots"
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/crypto/format"
	"gitlab.com/privategrity/crypto/forward"
	"math"
	"time"
)

// APIMessages are an implementation of the format.Message interface that's
// easy to use from Go
type APIMessage struct {
	Payload     string
	SenderID    user.ID
	RecipientID user.ID
}

func (m APIMessage) GetSender() []byte {
	return m.SenderID.Bytes()
}

func (m APIMessage) GetRecipient() []byte {
	return m.RecipientID.Bytes()
}

func (m APIMessage) GetPayload() string {
	return m.Payload
}

// Populates a text message and returns its wire representation
// TODO support multi-type messages or telling if a message is too long?
func FormatTextMessage(message string) []byte {
	textMessage := parse.TextMessage{
		Order: &parse.RepeatedOrdering{
			Time:       time.Now().Unix(),
			ChunkIndex: 0,
			Length:     1,
		},
		Display: &parse.DisplayData{
			Color: 0,
		},
		Message: message,
	}
	wireRepresentation, _ := proto.Marshal(&textMessage)
	return parse.Pack(&parse.TypedBody{
		Type: 1,
		Body: wireRepresentation,
	})
}

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
// returns in error if it fails
func InitClient(s globals.Storage, loc string) error {
	storageErr := globals.InitStorage(s, loc)

	if storageErr != nil {
		storageErr = errors.New(
			"could not init client storage: " + storageErr.Error())
		return storageErr
	}

	crypto.InitCrypto()

	return nil
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func Register(registrationCode string, gwAddr string,
	numNodes uint) (user.ID, error) {

	var err error

	if numNodes < 1 {
		jww.ERROR.Printf("Register: Invalid number of nodes")
		err = errors.New("could not register due to invalid number of nodes")
		return 0, err
	}

	hashUID := cyclic.NewIntFromString(registrationCode, 32).Bytes()
	UID, successLook := user.Users.LookupUser(string(hashUID))
	defer clearUserID(&UID)

	if !successLook {
		jww.ERROR.Printf("Register: HUID does not match")
		err = errors.New("could not register due to invalid HUID")
		return 0, err
	}

	u, successGet := user.Users.GetUser(UID)

	if !successGet {
		jww.ERROR.Printf("Register: ID lookup failed")
		err = errors.New("could not register due to ID lookup failure")
		return 0, err
	}

	nodekeys, successKeys := user.Users.LookupKeys(u.UserID)
	nodekeys.PublicKey = cyclic.NewInt(0)

	if !successKeys {
		jww.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return 0, err
	}

	nk := make([]user.NodeKeys, numNodes)

	for i := uint(0); i < numNodes; i++ {
		nk[i] = *nodekeys
	}

	nus := user.NewSession(u, gwAddr, nk)

	errStore := nus.StoreSession()

	if errStore != nil {
		err = errors.New(fmt.Sprintf(
			"Register: could not register due to failed session save"+
				": %s", errStore.Error()))
		jww.ERROR.Printf(err.Error())
		return 0, err
	}

	nus.Immolate()
	nus = nil

	return UID, err
}

// Logs in user and returns their nickname.
// returns an empty sting if login fails.
func Login(UID user.ID, addr string) (string, error) {

	err := user.LoadSession(UID)

	if user.TheSession == nil {
		return "", errors.New("Unable to load session")
	}

	if addr != "" {
		user.TheSession.SetGWAddress(addr)
	}

	addrToUse := user.TheSession.GetGWAddress()

	// TODO: These can be separate, but we set them to the same thing
	//       until registration is completed.
	io.SendAddress = addrToUse
	io.ReceiveAddress = addrToUse

	if err != nil {
		err = errors.New(fmt.Sprintf("Login: Could not login: %s",
			err.Error()))
		jww.ERROR.Printf(err.Error())
		return "", err
	}

	pollWaitTimeMillis := 1000 * time.Millisecond
	// FIXME listenCh won't exist - how do you tell if the reception thread
	// is running?
	if listenCh == nil {
		go io.Messaging.MessageReceiver(pollWaitTimeMillis)
	} else {
		jww.ERROR.Printf("Message receiver already started!")
	}

	return user.TheSession.GetCurrentUser().Nick, nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func Send(message format.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := user.ID(cyclic.NewIntFromBytes(message.
		GetRecipient()).Uint64())
	err := io.Messaging.SendMessage(recipientID, message.GetPayload())
	return err
}

// DisableBlockingTransmission turns off blocking transmission, for
// use with the channel bot and dummy bot
func DisableBlockingTransmission() {
	io.BlockTransmissions = false
}

// SetRateLimiting sets the minimum amount of time between message
// transmissions just for testing, probably to be removed in production
func SetRateLimiting(limit uint32) {
	io.TransmitDelay = time.Duration(limit) * time.Millisecond
}

// FIXME there can only be one
var listenCh chan *format.Message

func Listen(user user.ID, messageType parse.Type,
	newListener switchboard.Listener) {
	jww.INFO.Printf("Listening now: user %v, message type %v, ",
		user, messageType)
	switchboard.Listeners.Register(user, messageType, newListener)
}

type APISender struct{}

func (s APISender) Send(messageInterface format.MessageInterface) {
	Send(messageInterface)
}

type Sender interface {
	Send(messageInterface format.MessageInterface)
}

// Logout closes the connection to the server at this time and does
// nothing with the user id. In the future this will release resources
// and safely release any sensitive memory.
func Logout() error {
	if user.TheSession == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		jww.ERROR.Printf(err.Error())
		return err
	}

	io.Disconnect(io.SendAddress)
	if io.SendAddress != io.ReceiveAddress {
		io.Disconnect(io.ReceiveAddress)
	}

	errStore := user.TheSession.StoreSession()

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := user.TheSession.Immolate()

	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	return nil
}

func GetContactList() ([]user.ID, []string) {
	return user.Users.GetContactList()
}

func clearUserID(u *user.ID) {
	*u = math.MaxUint64
	*u = 0
}

func DisableRatchet() {
	forward.SetRatchetStatus(false)
}

func RegisterForUserDiscovery(emailAddress string) error {
	valueType := "EMAIL"
	userExists, err := bots.Search(valueType, emailAddress)
	if userExists != nil {
		jww.DEBUG.Printf("Already registered %s", emailAddress)
		return nil
	}
	if err != nil {
		return err
	}

	publicKey := user.TheSession.GetPublicKey()
	// Does cyclic do auto-pad? probably not...
	publicKeyBytes := publicKey.Bytes()
	fixedPubBytes := make([]byte, 256)
	for i := range publicKeyBytes {
		idx := len(fixedPubBytes) - i - 1
		if idx < 0 {
			jww.FATAL.Panicf("pubkey exceeds 2048 bit length!")
		}
		fixedPubBytes[idx] = publicKeyBytes[idx]
	}
	return bots.Register(valueType, emailAddress, fixedPubBytes)
}

func SearchForUser(emailAddress string) (map[uint64][]byte, error) {
	valueType := "EMAIL"
	return bots.Search(valueType, emailAddress)
}
