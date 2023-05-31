////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/restlike"
	"gitlab.com/elixxir/client/v4/restlike/connect"
)

// RestlikeMessage is the bindings' representation of a restlike.Message
//
// JSON example:
//
//	{
//	 "Version":1,
//	 "Headers":"Y29udGVudHM6YXBwbGljYXRpb24vanNvbg==",
//	 "Content":"VGhpcyBpcyBhIHJlc3RsaWtlIG1lc3NhZ2U=",
//	 "Method":2,
//	 "URI":"xx://CmixRestlike/rest",
//	 "Error":""
//	}
type RestlikeMessage struct {
	Version uint32
	Headers []byte
	Content []byte
	Method  int
	URI     string
	Error   string
}

// RestlikeRequest performs a normal restlike request.
//
// Parameters:
//   - cmixId - ID of the cMix object in the tracker
//   - connectionID - ID of the connection in the tracker
//   - request - JSON marshalled RestlikeMessage
//   - e2eParamsJSON - JSON marshalled xxdk.E2EParams
//
// Returns:
//   - []byte - JSON marshalled RestlikeMessage
func RestlikeRequest(
	cmixId, connectionID int, request, e2eParamsJSON []byte) ([]byte, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("restlike params unspecified, using defaults")
		e2eParamsJSON = GetDefaultE2EParams()
	}

	cl, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}
	conn, err := connectionTrackerSingleton.get(connectionID)
	if err != nil {
		return nil, err
	}

	params, err := parseE2EParams(e2eParamsJSON)
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
	}, params.Base)
	if err != nil {
		return nil, err
	}

	respMessage := &RestlikeMessage{
		Version: result.Headers.Version,
		Headers: result.Headers.Headers,
		Content: result.Content,
		Method:  int(result.Method),
		URI:     result.Uri,
		Error:   result.Error,
	}
	return json.Marshal(respMessage)
}

// RestlikeRequestAuth performs an authenticated restlike request.
//
// Parameters:
//   - cmixId - ID of the cMix object in the tracker
//   - authConnectionID - ID of the authenticated connection in the tracker
//   - request - JSON marshalled RestlikeMessage
//   - e2eParamsJSON - JSON marshalled xxdk.E2EParams
//
// Returns:
//   - []byte - JSON marshalled RestlikeMessage
func RestlikeRequestAuth(cmixId, authConnectionID int, request,
	e2eParamsJSON []byte) ([]byte, error) {
	if len(e2eParamsJSON) == 0 {
		jww.WARN.Printf("restlike params unspecified, using defaults")
		e2eParamsJSON = GetDefaultE2EParams()
	}
	params, err := parseE2EParams(e2eParamsJSON)
	if err != nil {
		return nil, err
	}

	cl, err := cmixTrackerSingleton.get(cmixId)
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

	result, err := c.Request(restlike.Method(msg.Method), restlike.URI(msg.URI),
		msg.Content, &restlike.Headers{
			Headers: msg.Headers,
			Version: msg.Version,
		}, params.Base)
	if err != nil {
		return nil, err
	}
	respMessage := &RestlikeMessage{
		Version: result.Headers.Version,
		Headers: result.Headers.Headers,
		Content: result.Content,
		Method:  int(result.Method),
		URI:     result.Uri,
		Error:   result.Error,
	}
	return json.Marshal(respMessage)
}
