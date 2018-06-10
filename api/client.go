////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package api

import (
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/bots"
	"gitlab.com/privategrity/client/crypto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/parse"
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
	SenderID    uint64
	RecipientID uint64
}

func (m APIMessage) GetSender() []byte {
	senderAsInt := cyclic.NewIntFromUInt(m.SenderID)
	return senderAsInt.LeftpadBytes(format.SID_LEN)
}

func (m APIMessage) GetRecipient() []byte {
	recipientAsInt := cyclic.NewIntFromUInt(m.RecipientID)
	return recipientAsInt.LeftpadBytes(format.RID_LEN)
}

func (m APIMessage) GetPayload() string {
	return m.Payload
}

// Initializes the client by registering a storage mechanism.
// If none is provided, the system defaults to using OS file access
// returns in error if it fails
func InitClient(s globals.Storage, loc string, receiver globals.Receiver) error {
	storageErr := globals.InitStorage(s, loc)

	if storageErr != nil {
		storageErr = errors.New(
			"could not init client storage: " + storageErr.Error())
		return storageErr
	}

	crypto.InitCrypto()

	receiverErr := globals.SetReceiver(receiver)

	if receiverErr != nil {
		return receiverErr
	}

	return nil
}

// Registers user and returns the User ID.
// Returns an error if registration fails.
func Register(registrationCode uint64, gwAddr string,
	numNodes uint) (uint64, error) {

	var err error

	if numNodes < 1 {
		jww.ERROR.Printf("Register: Invalid number of nodes")
		err = errors.New("could not register due to invalid number of nodes")
		return 0, err
	}

	UID, successLook := globals.Users.LookupUser(registrationCode)
	defer clearUint64(&UID)

	if !successLook {
		jww.ERROR.Printf("Register: HUID does not match")
		err = errors.New("could not register due to invalid HUID")
		return 0, err
	}

	user, successGet := globals.Users.GetUser(UID)

	if !successGet {
		jww.ERROR.Printf("Register: UserID lookup failed")
		err = errors.New("could not register due to UserID lookup failure")
		return 0, err
	}

	nodekeys, successKeys := globals.Users.LookupKeys(user.UserID)
	nodekeys.PublicKey = cyclic.NewInt(0)

	if !successKeys {
		jww.ERROR.Printf("Register: could not find user keys")
		err = errors.New("could not register due to missing user keys")
		return 0, err
	}

	nk := make([]globals.NodeKeys, numNodes)

	for i := uint(0); i < numNodes; i++ {
		nk[i] = *nodekeys
	}

	nus := globals.NewUserSession(user, gwAddr, nk)

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
func Login(UID uint64, addr string) (string, error) {

	err := globals.LoadSession(UID)

	if globals.Session == nil {
		return "", errors.New("Unable to load session")
	}

	if addr != "" {
		globals.Session.SetGWAddress(addr)
	}

	addrToUse := globals.Session.GetGWAddress()

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
	if listenCh == nil {
		listenCh = io.Messaging.Listen(0)
		go io.Messaging.MessageReceiver(pollWaitTimeMillis)
	} else {
		jww.ERROR.Printf("Message receiver already started!")
	}

	return globals.Session.GetCurrentUser().Nick, nil
}

// Send prepares and sends a message to the cMix network
// FIXME: We need to think through the message interface part.
func Send(message format.MessageInterface) error {
	// FIXME: There should (at least) be a version of this that takes a byte array
	recipientID := cyclic.NewIntFromBytes(message.GetRecipient()).Uint64()
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

var listenCh chan *format.Message

// TryReceive checks if there is a received message on the internal fifo.
// returns nil if there isn't.
// FIXME: There's not a good reason to return an error here. I nil'd it out
// for now but it should be removed. Before it was returning an error only if
// the user had not been logged in yet.
func TryReceive() (format.MessageInterface, error) {
	select {
	case message := <-listenCh:
		var m APIMessage
		if message.GetPayload() != "" {
			// FIXME: Post-refactor, it would mak emore sense to
			// have a channel bot listener that populates the channel
			// bot messages, and to ignore the channelbot messages in this
			// loop or to try to get & parse them a different way.
			// try to parse the gob (in case it's from a channel)
			channelMessage, err := parse.ParseChannelbotMessage(
				message.GetPayload())
			if err == nil {
				// Message from channelbot
				// TODO Speaker ID has been hacked into the Recipient ID
				// for channels
				m.SenderID = message.GetSenderIDUint()
				m.Payload = channelMessage.Message
				m.RecipientID = channelMessage.SpeakerID
			} else {
				// Message from normal client
				m.SenderID = message.GetSenderIDUint()
				m.Payload = message.GetPayload()
				m.RecipientID = message.GetRecipientIDUint()
			}
		}
		return m, nil
	default:
		// Would want to return nil here, but callers are used to this
		return &APIMessage{SenderID: 0, Payload: "", RecipientID: 0}, nil
	}
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
	if globals.Session == nil {
		err := errors.New("Logout: Cannot Logout when you are not logged in")
		jww.ERROR.Printf(err.Error())
		return err
	}

	io.Disconnect(io.SendAddress)
	if io.SendAddress != io.ReceiveAddress {
		io.Disconnect(io.ReceiveAddress)
	}

	errStore := globals.Session.StoreSession()

	if errStore != nil {
		err := errors.New(fmt.Sprintf("Logout: Store Failed: %s" +
			errStore.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	errImmolate := globals.Session.Immolate()

	if errImmolate != nil {
		err := errors.New(fmt.Sprintf("Logout: Immolation Failed: %s" +
			errImmolate.Error()))
		jww.ERROR.Printf(err.Error())
		return err
	}

	return nil
}

func GetContactList() ([]uint64, []string) {
	return globals.Users.GetContactList()
}

func clearUint64(u *uint64) {
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

	publicKey := globals.Session.GetPublicKey()
	// Does cyclic do auto-pad? probably not...
	publicKeyBytes := publicKey.Bytes()
	fixedPubBytes := make([]byte, 256)
	for i := range publicKeyBytes {
		idx := len(fixedPubBytes) - i - 1
		if idx < 0 {
			jww.ERROR.Printf("Trimming pubkey because it exceeds 2048 bit length!")
			break
		}
		fixedPubBytes[idx] = publicKeyBytes[idx]
	}
	return bots.Register(valueType, emailAddress, fixedPubBytes)
}

func SearchForUser(emailAddress string) (map[uint64][]byte, error) {
	valueType := "EMAIL"
	return bots.Search(valueType, emailAddress)
}
