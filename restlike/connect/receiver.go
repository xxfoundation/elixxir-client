////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/connect"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/restlike"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/xx_network/primitives/id"
	"google.golang.org/protobuf/proto"
)

// receiver is the reception handler for a RestServer
type receiver struct {
	conn      connect.Connection
	endpoints *restlike.Endpoints
	net       *xxdk.Cmix
}

// Hear handles connect.Connection message reception for a RestServer
// Automatically responds to invalid endpoint requests
func (c receiver) Hear(item receive.Message) {
	// Unmarshal the request payload
	newMessage := &restlike.Message{}
	err := proto.Unmarshal(item.Payload, newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	var respondErr error
	uri := restlike.URI(newMessage.GetUri())
	method := restlike.Method(newMessage.GetMethod())
	if cb, err := c.endpoints.Get(uri, method); err == nil {
		// Send the payload to the proper Callback if it exists and
		// singleRespond with the result
		respondErr = respond(cb(newMessage), c.conn, c.net)
	} else {
		// If no callback, automatically send an error response
		respondErr = respond(
			&restlike.Message{Error: err.Error()}, c.conn, c.net)
	}
	if respondErr != nil {
		jww.ERROR.Printf("Unable to singleRespond to request: %+v", err)
	}
}

// respond to connect.Connection with the given Message
func respond(
	response *restlike.Message, conn connect.Connection, net *xxdk.Cmix) error {
	payload, err := proto.Marshal(response)
	if err != nil {
		return errors.Errorf("unable to marshal restlike response message: %+v", err)
	}

	// TODO: Parameterize params
	for {
		sendReport, err := conn.SendE2E(catalog.XxMessage, payload,
			e2e.GetDefaultParams())
		if err != nil {
			return errors.Errorf("unable to send restlike response message: %+v", err)
		}
		if !verifySendSuccess(net, e2e.GetDefaultParams(),
			sendReport.RoundList, conn.GetPartner().PartnerId(), payload) {
			continue
		}

		break
	}

	_, err = conn.SendE2E(catalog.XxMessage, payload, e2e.GetDefaultParams())
	if err != nil {
		return errors.Errorf("unable to send restlike response message: %+v", err)
	}

	return nil
}

// Name is used for debugging
func (c receiver) Name() string {
	return "Restlike"
}

func verifySendSuccess(user *xxdk.Cmix, paramsE2E e2e.Params,
	roundIDs []id.Round, partnerId *id.ID, payload []byte) bool {
	retryChan := make(chan struct{})
	done := make(chan struct{}, 1)

	// Construct the callback function which
	// verifies successful message send or retries
	f := func(allRoundsSucceeded, timedOut bool,
		rounds map[id.Round]cmix.RoundResult) {
		if !allRoundsSucceeded {
			retryChan <- struct{}{}
		} else {
			done <- struct{}{}
		}
	}

	// Monitor rounds for results
	user.GetCmix().GetRoundResults(
		paramsE2E.CMIXParams.Timeout, f, roundIDs...)

	select {
	case <-retryChan:
		// On a retry, go to the top of the loop
		jww.DEBUG.Printf("Messages were not sent successfully," +
			" resending messages...")
		return false
	case <-done:
		// Close channels on verification success
		close(done)
		close(retryChan)
		return true
	}
}
