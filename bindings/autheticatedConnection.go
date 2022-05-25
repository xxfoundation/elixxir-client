package bindings

import (
	"gitlab.com/elixxir/client/connect"
	"gitlab.com/elixxir/crypto/contact"
)

//connection tracker singleton, used to track connections so they can be
//referenced by id back over the bindings
var authenticatedConnectionTrackerSingleton = &authenticatedConnectionTracker{
	connections: make(map[int]*AuthenticatedConnection),
	count:       0,
}

type AuthenticatedConnection struct {
	Connection
}

func (_ *AuthenticatedConnection) IsAuthenticated() bool {
	return true
}

// ConnectWithAuthentication is called by the client, ie the one establishing
// connection with the server. Once a connect.Connection has been established
// with the server and then authenticate their identity to the server.
func (c *Client) ConnectWithAuthentication(recipientContact []byte, myIdentity []byte) (*AuthenticatedConnection, error) {
	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}
	myID, rsaPriv, salt, myDHPriv, err := c.unmarshalIdentity(myIdentity)
	if err != nil {
		return nil, err
	}

	connection, err := connect.ConnectWithAuthentication(cont, myID, salt, rsaPriv, myDHPriv, c.api.GetRng(),
		c.api.GetStorage().GetE2EGroup(), c.api.GetCmix(), connect.GetDefaultParams())
	return authenticatedConnectionTrackerSingleton.make(connection), nil
}
