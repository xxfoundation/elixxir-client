package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	e2e2 "gitlab.com/elixxir/client/e2e"
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
// Example E2ESendReport:
// {"RoundList":{"Rounds":[1,5,9]},
//  "MessageID":"51Yy47uZbP0o2Y9B/kkreDLTB6opUol3M3mYiY2dcdQ=",
//  "Timestamp":1653582683183384000}
type E2ESendReport struct {
	RoundsList
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

	sr.RoundsList = makeRoundsList(rounds)

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

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
// Returns marshalled ListenerID
func (c *Connection) RegisterListener(messageType int, newListener Listener) {
	_ = c.connection.RegisterListener(catalog.MessageType(messageType), listener{l: newListener})
}
