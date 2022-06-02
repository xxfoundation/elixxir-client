package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/client/restlike/single"
	singleuse "gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
)

// RestlikeCallback is the public function type bindings can use to make an asynchronous restlike request
// It accepts a json marshalled restlike.Message and an error (the results of calling json.Marshal on the message)
type RestlikeCallback interface {
	Callback([]byte, error)
}

// RequestRestLike sends a restlike request to a given contact
// Accepts marshalled contact object as recipient, byte slice payload & headers, method enum and a URI
// Returns json marshalled restlike.Message & error
func RequestRestLike(clientID int, recipient, request []byte) ([]byte, error) {
	c, err := clientTrackerSingleton.get(clientID)
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

	resp, err := req.Request(recipientContact, restlike.Method(message.Method), restlike.URI(message.URI),
		message.Content, &restlike.Headers{
			Headers: message.Headers,
			Version: 0,
		}, singleuse.GetDefaultRequestParams())
	if err != nil {
		return nil, err
	}
	return json.Marshal(resp)
}

// AsyncRequestRestLike sends an asynchronous restlike request to a given contact
// Accepts marshalled contact object as recipient, byte slice payload & headers, method enum, URI, and a RestlikeCallback
// Returns an error, and the RestlikeCallback will be called with the results of json marshalling the response when received
func AsyncRequestRestLike(clientID int, recipient, request []byte, cb RestlikeCallback) error {
	c, err := clientTrackerSingleton.get(clientID)
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

	return req.AsyncRequest(recipientContact, restlike.Method(message.Method), restlike.URI(message.URI),
		message.Content, &restlike.Headers{
			Headers: message.Headers,
			Version: 0,
		}, rlcb, singleuse.GetDefaultRequestParams())
}
