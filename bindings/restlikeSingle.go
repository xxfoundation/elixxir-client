///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"

	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/client/restlike/single"
	"gitlab.com/elixxir/crypto/contact"
)

// RestlikeCallback is the public function type bindings can use to make an asynchronous restlike request
// It accepts a json marshalled restlike.Message and an error (the results of calling json.Marshal on the message)
type RestlikeCallback interface {
	Callback([]byte, error)
}

// RequestRestLike sends a restlike request to a given contact
// Accepts marshalled contact object as recipient, marshalled RestlikeMessage and params JSON
// Returns json marshalled restlike.Message & error
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

// AsyncRequestRestLike sends an asynchronous restlike request to a given contact
// Accepts e2e client ID, marshalled contact object as recipient,
// marshalled RestlikeMessage, marshalled Params json, and a RestlikeCallback
// Returns an error, and the RestlikeCallback will be called with the results
// of json marshalling the response when received
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

	return req.AsyncRequest(recipientContact, restlike.Method(message.Method), restlike.URI(message.URI),
		message.Content, &restlike.Headers{
			Headers: message.Headers,
			Version: 0,
		}, rlcb, params)
}
