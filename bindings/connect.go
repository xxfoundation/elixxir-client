package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	e2e2 "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/crypto/contact"
)

//connection tracker singleton, used to track connections so they can be
//referenced by id back over the bindings
var connectionTrackerSingleton = &connectionTracker{
	connections: make(map[int]*Connection),
	count:       0,
}

//
type Connection struct {
	connection connect.Connection
	id         int
}

// Connect performs auth key negotiation with the given recipient,
// and returns a Connection object for the newly-created partner.Manager
// This function is to be used sender-side and will block until the
// partner.Manager is confirmed.
func (c *Client) Connect(recipientContact []byte, myIdentity []byte) (
	*Connection, error) {
	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}
	myID, _, _, myDHPriv, err := unmarshalIdentity(myIdentity)
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

//
type E2ESendReport struct {
	roundsList
	MessageID []byte
	Timestamp int64
}

// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
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

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
func (c *Connection) RegisterListener(messageType int, newListener receive.Listener) receive.ListenerID {

}

// Unregister listener for E2E reception
func (c *Connection) Unregister(listenerID receive.ListenerID) {

}
