////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/client/restlike/connect"
)

//
func RestlikeRequest(clientID int, connectionID int, request []byte) ([]byte, error) {
	cl, err := clientTrackerSingleton.get(clientID)
	if err != nil {
		return nil, err
	}
	conn, err := connectionTrackerSingleton.get(connectionID)
	if err != nil {
		return nil, err
	}

	msg := &RestlikeMessage{}
	err = json.Unmarshal(request, msg)
	if err != nil {
		return nil, err
	}

	c := connect.Request{
		Net:    conn.connection,
		Rng:    cl.api.GetRng().GetStream(),
		E2eGrp: nil,
	}

	result, err := c.Request(restlike.Method(msg.Method), restlike.URI(msg.URI), msg.Content, &restlike.Headers{
		Headers: msg.Headers,
		Version: msg.Version,
	}, e2e.GetDefaultParams())
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}

//
type RestlikeMessage struct {
	Version uint32
	Headers []byte
	Content []byte
	Method  int
	URI     string
	Error   string
}

//
func RestlikeRequestAuth(clientID int, authConnectionID int, request []byte) ([]byte, error) {
	cl, err := clientTrackerSingleton.get(clientID)
	if err != nil {
		return nil, err
	}
	auth, err := authenticatedConnectionTrackerSingleton.get(authConnectionID)
	if err != nil {
		return nil, err
	}

	msg := &RestlikeMessage{}
	err = json.Unmarshal(request, msg)
	if err != nil {
		return nil, err
	}

	c := connect.Request{
		Net:    auth.connection,
		Rng:    cl.api.GetRng().GetStream(),
		E2eGrp: nil,
	}

	result, err := c.Request(restlike.Method(msg.Method), restlike.URI(msg.URI), msg.Content, &restlike.Headers{
		Headers: msg.Headers,
		Version: msg.Version,
	}, e2e.GetDefaultParams())
	if err != nil {
		return nil, err
	}
	return json.Marshal(result)
}
