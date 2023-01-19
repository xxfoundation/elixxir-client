////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"

	"gitlab.com/elixxir/client/v4/restlike"
	"gitlab.com/elixxir/client/v4/restlike/single"
	"gitlab.com/elixxir/crypto/contact"
)

// RestlikeCallback is the public function type bindings can use to make an
// asynchronous restlike request.
//
// Parameters:
//  - []byte - JSON marshalled restlike.Message
//  - error - an error (the results of calling json.Marshal on the message)
type RestlikeCallback interface {
	Callback([]byte, error)
}

// RequestRestLike sends a restlike request to a given contact.
//
// Parameters:
//  - e2eID - ID of the e2e object in the tracker
//  - recipient - marshalled contact.Contact object
//  - request - JSON marshalled RestlikeMessage
//  - paramsJSON - JSON marshalled single.RequestParams
//
// Returns:
//  - []byte - JSON marshalled restlike.Message
func RequestRestLike(e2eID int, recipient, request, paramsJSON []byte) ([]byte, error) {
	c, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}
	req := single.Request{
		Net:    c.api.GetCmix(),
		Rng:    c.api.GetRng().GetStream(),
		E2eGrp: c.api.GetStorage().GetE2EGroup(),
	}

	message := &RestlikeMessage{}
	err = json.Unmarshal(request, message)

	recipientContact, err := contact.Unmarshal(recipient)
	if err != nil {
		return nil, err
	}

	params, err := parseSingleUseParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	resp, err := req.Request(recipientContact, restlike.Method(message.Method), restlike.URI(message.URI),
		message.Content, &restlike.Headers{
			Headers: message.Headers,
			Version: 0,
		}, params)
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp)
}

// AsyncRequestRestLike sends an asynchronous restlike request to a given
// contact.
//
// Parameters:
//  - e2eID - ID of the e2e object in the tracker
//  - recipient - marshalled contact.Contact object
//  - request - JSON marshalled RestlikeMessage
//  - paramsJSON - JSON marshalled single.RequestParams
//  - cb - RestlikeCallback callback
//
// Returns an error, and the RestlikeCallback will be called with the results
// of JSON marshalling the response when received.
func AsyncRequestRestLike(e2eID int, recipient, request, paramsJSON []byte, cb RestlikeCallback) error {
	c, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return err
	}
	req := single.Request{
		Net:    c.api.GetCmix(),
		Rng:    c.api.GetRng().GetStream(),
		E2eGrp: c.api.GetStorage().GetE2EGroup(),
	}

	message := &RestlikeMessage{}
	err = json.Unmarshal(request, message)

	recipientContact, err := contact.Unmarshal(recipient)
	if err != nil {
		return err
	}

	rlcb := func(message *restlike.Message) {
		cb.Callback(json.Marshal(message))
	}

	params, err := parseSingleUseParams(paramsJSON)
	if err != nil {
		return err
	}

	return req.AsyncRequest(
		recipientContact, restlike.Method(message.Method),
		restlike.URI(message.URI), message.Content, &restlike.Headers{
			Headers: message.Headers,
			Version: 0,
		}, rlcb, params)
}
