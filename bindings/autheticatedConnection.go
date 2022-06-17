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

// ConnectWithAuthentication is called by the client (i.e. the one establishing
// connection with the server). Once a connect.Connection has been established
// with the server and then authenticate their identity to the server.
// accepts a marshalled TransmissionIdentity and contact.Contact object
func (c *Cmix) ConnectWithAuthentication(e2eId int, recipientContact []byte) (*AuthenticatedConnection, error) {
	cont, err := contact.Unmarshal(recipientContact)
	if err != nil {
		return nil, err
	}

	e2eClient, err := e2eTrackerSingleton.get(e2eId)
	if err != nil {
		return nil, err
	}

	connection, err := connect.ConnectWithAuthentication(cont, e2eClient.api, connect.GetDefaultParams())
	return authenticatedConnectionTrackerSingleton.make(connection), nil
}
