package bindings

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	e2e2 "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/crypto/contact"
)

// connectionTrackerSingleton is used to track connections so they can be
// referenced by id back over the bindings
var connectionTrackerSingleton = &connectionTracker{
	connections: make(map[int]*Connection),
	count:       0,
}

// Connection is the bindings representation of a connect.Connection object that can be tracked
type Connection struct {
	connection connect.Connection
	id         int
}

// Connect performs auth key negotiation with the given recipient,
// and returns a Connection object for the newly-created partner.Manager
// This function is to be used sender-side and will block until the
// partner.Manager is confirmed.
// recipientContact - marshalled contact.Contact object
// myIdentity - marshalled Identity object
func (c *Client) Connect(recipientContact []byte, myIdentity []byte) (
	*Connection, error) {
	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}
	myID, _, _, myDHPriv, err := c.unmarshalIdentity(myIdentity)
	if err != nil {
		return nil, err
	}

	connection, err := connect.Connect(cont, myID, myDHPriv, c.api.GetRng(),
		c.api.GetStorage().GetE2EGroup(), c.api.GetCmix(), connect.GetDefaultParams())

	if err != nil {
		return nil, err
	}

	return connectionTrackerSingleton.make(connection), nil
}

// E2ESendReport is the bindings representation of the return values of SendE2E
type E2ESendReport struct {
	roundsList
	MessageID []byte
	Timestamp int64
}

// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
// Returns marshalled E2ESendReport
func (c *Connection) SendE2E(mt int, payload []byte) ([]byte, error) {
	rounds, mid, ts, err := c.connection.SendE2E(catalog.MessageType(mt), payload,
		e2e2.GetDefaultParams())

	if err != nil {
		return nil, err
	}

	sr := E2ESendReport{
		MessageID: mid.Marshal(),
		Timestamp: ts.UnixNano(),
	}

	sr.roundsList = makeRoundsList(rounds)

	return json.Marshal(&sr)
}

// Close deletes this Connection's partner.Manager and releases resources
func (c *Connection) Close() {
	c.Close()
}

// GetPartner returns the partner.Manager for this Connection
func (c *Connection) GetPartner() []byte {
	return c.connection.GetPartner().PartnerId().Marshal()
}

// Listener provides a callback to hear a message
// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	// Hear is called to receive a message in the UI
	Hear(item []byte)
	// Returns a name, used for debugging
	Name() string
}

//
type listener struct {
	l Listener
}

// Message is the bindings representation of a receive.Message
type Message struct {
	MessageType int
	ID          []byte
	Payload     []byte

	Sender      []byte
	RecipientID []byte
	EphemeralID int64
	Timestamp   int64 // Message timestamp of when the user sent

	Encrypted bool
	RoundId   int
}

// Hear is called to receive a message in the UI
func (l listener) Hear(item receive.Message) {
	m := Message{
		MessageType: int(item.MessageType),
		ID:          item.ID.Marshal(),
		Payload:     item.Payload,
		Sender:      item.Sender.Marshal(),
		RecipientID: item.RecipientID.Marshal(),
		EphemeralID: item.EphemeralID.Int64(),
		Timestamp:   item.Timestamp.UnixNano(),
		Encrypted:   item.Encrypted,
		RoundId:     int(item.Round.ID),
	}
	result, err := json.Marshal(&m)
	if err != nil {
		jww.ERROR.Printf("Unable to marshal Message: %+v", err.Error())
	}
	l.l.Hear(result)
}

// Name used for debugging
func (l listener) Name() string {
	return l.l.Name()
}

// ListenerID represents the return type of RegisterListener
type ListenerID struct {
	userID      []byte
	messageType int
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
// Returns marshalled ListenerID
func (c *Connection) RegisterListener(messageType int, newListener Listener) []byte {
	listenerId := c.connection.RegisterListener(catalog.MessageType(messageType), listener{l: newListener})
	newlistenerId := ListenerID{
		userID:      listenerId.GetUserID().Marshal(),
		messageType: int(listenerId.GetMessageType()),
	}
	result, err := json.Marshal(&newlistenerId)
	if err != nil {
		jww.ERROR.Printf("Unable to marshal listenerId: %+v", err.Error())
	}
	return result
}
