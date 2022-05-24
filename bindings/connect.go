package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/connect"
	e2e2 "gitlab.com/elixxir/client/e2e"
	contact2 "gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

//connection tracker singleton, used to track connections so they can be
//referenced by id back over the bindings
var connectionTrackerSingleton = &connectionTracker{
	connections: make(map[int]Connection),
	count:       0,
}

type Connection struct {
	connection connect.Connection
	id         int
}

// Connect blocks until it connects to the remote
func (c *Client) Connect(recipientContact []byte, myIdentity []byte) (
	*Connection, error) {
	cont, err := contact2.Unmarshal(recipientContact)
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

type E2ESendReport struct {
	roundsList
	MessageID []byte
	Timestamp int64
}

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
