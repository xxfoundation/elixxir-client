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

// RestlikeMessage is the bindings representation of a restlike.Message
// Example marshalled RestlikeMessage:
//{"Version":1,
// "Headers":"Y29udGVudHM6YXBwbGljYXRpb24vanNvbg==",
// "Content":"VGhpcyBpcyBhIHJlc3RsaWtlIG1lc3NhZ2U=",
// "Method":2,
// "URI":"xx://CmixRestlike/rest",
// "Error":""}
type RestlikeMessage struct {
	Version uint32
	Headers []byte
	Content []byte
	Method  int
	URI     string
	Error   string
}

// RestlikeRequest performs a normal restlike request
// request - marshalled RestlikeMessage
// Returns marshalled result RestlikeMessage
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

// RestlikeRequestAuth performs an authenticated restlike request
// request - marshalled RestlikeMessage
// Returns marshalled result RestlikeMessage
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
