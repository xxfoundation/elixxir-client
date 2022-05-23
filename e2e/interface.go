package e2e

import (
	"time"

	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
)

type Handler interface {
	// StartProcesses - process control which starts the running of rekey
	// handlers and the critical message handlers
	StartProcesses() (stoppable.Stoppable, error)

	// SendE2E send a message containing the payload to the
	// recipient of the passed message type, per the given
	// parameters - encrypted with end to end encryption.
	// Default parameters can be retrieved through
	// GetDefaultParams()
	// If too long, it will chunk a message up into its messages
	// and send each as a separate cmix message. It will return
	// the list of all rounds sent on, a unique ID for the
	// message, and the timestamp sent on.
	// the recipient must already have an e2e relationship,
	// otherwise an error will be returned.
	// Will return an error if the network is not healthy or in
	// the event of a failed send
	SendE2E(mt catalog.MessageType, recipient *id.ID, payload []byte,
		params Params) ([]id.Round, e2e.MessageID, time.Time, error)

	/* === Reception ==================================================== */

	// RegisterListener Registers a new listener. Returns the ID
	// of the new listener. Keep the ID around if you want to be
	// able to delete the listener later.
	//
	// The name is used for debug printing and not checked for
	// uniqueness
	//
	// user: id.ZeroUser for all, or any user ID to listen for
	// messages from a particular user.
	// messageType: catalog.NoType for all, or any message type to
	// listen for messages of that type.
	// newListener: something implementing the Listener
	// interface. Do not pass nil to this.
	//
	// If a message matches multiple listeners, all of them will
	// hear the message.
	RegisterListener(senderID *id.ID,
		messageType catalog.MessageType,
		newListener receive.Listener) receive.ListenerID

	// RegisterFunc Registers a new listener built around the
	// passed function.  Returns the ID of the new listener. Keep
	// the ID around if you want to be able to delete the listener
	// later.
	//
	// name is used for debug printing and not checked for
	// uniqueness
	//
	// user: id.ZeroUser for all, or any user ID to listen for
	// messages from a particular user.
	// messageType: catalog.NoType for all, or any message type to
	// listen for messages of that type.
	// newListener: a function implementing the ListenerFunc
	// function type.  Do not pass nil to this.
	//
	// If a message matches multiple listeners, all of them will
	// hear the message.
	RegisterFunc(name string, senderID *id.ID,
		messageType catalog.MessageType,
		newListener receive.ListenerFunc) receive.ListenerID

	// RegisterChannel Registers a new listener built around the
	// passed channel.  Returns the ID of the new listener. Keep
	// the ID around if you want to be able to delete the listener
	// later.
	//
	// name is used for debug printing and not checked for
	// uniqueness
	//
	// user: 0 for all, or any user ID to listen for messages from
	// a particular user. 0 can be id.ZeroUser or id.ZeroID
	// messageType: 0 for all, or any message type to listen for
	// messages of that type. 0 can be Message.AnyType
	// newListener: an item channel.  Do not pass nil to this.
	//
	// If a message matches multiple listeners, all of them will
	// hear the message.
	RegisterChannel(name string, senderID *id.ID,
		messageType catalog.MessageType,
		newListener chan receive.Message) receive.ListenerID

	// Unregister removes the listener with the specified ID so it
	// will no longer get called
	Unregister(listenerID receive.ListenerID)

	/* === Partners ===================================================== */

	// AddPartner adds a partner. Automatically creates both send
	// and receive sessions using the passed cryptographic data
	// and per the parameters sent
	AddPartner(partnerID *id.ID,
		partnerPubKey, myPrivKey *cyclic.Int,
		partnerSIDHPubKey *sidh.PublicKey,
		mySIDHPrivKey *sidh.PrivateKey, sendParams,
		receiveParams session.Params) (partner.Manager, error)

	// GetPartner returns the partner per its ID, if it exists
	GetPartner(partnerID *id.ID) (partner.Manager, error)

	// DeletePartner removes the associated contact from the E2E store
	// myID is your ID in the relationship
	DeletePartner(partnerId *id.ID) error

	// GetAllPartnerIDs returns a list of all partner IDs that the user has
	// an E2E relationship with.
	GetAllPartnerIDs() []*id.ID

	/* === Services ===================================================== */

	// AddService adds a service for all partners of the given
	// tag, which will call back on the given processor. These can
	// be sent to using the tag fields in the Params Object
	// Passing nil for the processor allows you to create a
	// service which is never called but will be visible by
	// notifications. Processes added this way are generally not
	// end-to-end encrypted messages themselves, but other
	// protocols which piggyback on e2e relationships to start
	// communication
	AddService(tag string, processor message.Processor) error

	// RemoveService removes all services for the given tag
	RemoveService(tag string) error

	/* === Unsafe ======================================================= */

	// SendUnsafe sends a message without encryption. It breaks
	// both privacy and security. It does partition the
	// message. It should ONLY be used for debugging.
	// It does not respect service tags in the parameters and
	// sends all messages with "Silent" and "E2E" tags.
	// It does not support critical messages.
	// It does not check that an e2e relationship exists with the recipient
	// Will return an error if the network is not healthy or in the event of
	// a failed send
	SendUnsafe(mt catalog.MessageType, recipient *id.ID,
		payload []byte, params Params) ([]id.Round, time.Time, error)

	// EnableUnsafeReception enables the reception of unsafe message by
	// registering bespoke services for reception. For debugging only!
	EnableUnsafeReception()

	/* === Utility ====================================================== */

	// GetGroup returns the cyclic group used for end to end encruption
	GetGroup() *cyclic.Group

	// GetHistoricalDHPubkey returns the user's Historical DH
	// Public Key
	GetHistoricalDHPubkey() *cyclic.Int

	// GetHistoricalDHPrivkey returns the user's Historical DH Private Key
	GetHistoricalDHPrivkey() *cyclic.Int

	// GetReceptionID returns the default IDs
	GetReceptionID() *id.ID

	// FirstPartitionSize returns the max partition payload size for the
	// first payload
	FirstPartitionSize() uint

	// SecondPartitionSize returns the max partition payload size for all
	// payloads after the first payload
	SecondPartitionSize() uint

	// PartitionSize returns the partition payload size for the given
	// payload index. The first payload is index 0.
	PartitionSize(payloadIndex uint) uint

	// PayloadSize Returns the max payload size for a partitionable E2E
	// message
	PayloadSize() uint
}
