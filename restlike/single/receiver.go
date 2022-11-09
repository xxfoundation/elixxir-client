////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/restlike"
	"gitlab.com/elixxir/client/single"
	"google.golang.org/protobuf/proto"
	"time"
)

// receiver is the reception handler for a RestServer
type receiver struct {
	endpoints *restlike.Endpoints
}

// Callback is the handler for single-use message reception for a RestServer
// Automatically responds to invalid endpoint requests
func (s *receiver) Callback(req *single.Request,
	receptionId receptionID.EphemeralIdentity, rounds []rounds.Round) {
	// Unmarshal the request payload
	newMessage := &restlike.Message{}
	err := proto.Unmarshal(req.GetPayload(), newMessage)
	if err != nil {
		jww.ERROR.Printf("Unable to unmarshal restlike message: %+v", err)
		return
	}

	var respondErr error
	if cb, err := s.endpoints.Get(restlike.URI(newMessage.GetUri()),
		restlike.Method(newMessage.GetMethod())); err == nil {
		// Send the payload to the proper Callback if it exists and singleRespond with the result
		respondErr = singleRespond(cb(newMessage), req)
	} else {
		// If no callback, automatically send an error response
		respondErr = singleRespond(&restlike.Message{Error: err.Error()}, req)
	}
	if respondErr != nil {
		jww.ERROR.Printf("Unable to singleRespond to request: %+v", err)
	}
}

// singleRespond to a single.Request with the given Message
func singleRespond(response *restlike.Message, req *single.Request) error {
	payload, err := proto.Marshal(response)
	if err != nil {
		return errors.Errorf("unable to marshal restlike response message: %+v", err)
	}

	// TODO: Parameterize params and timeout
	_, err = req.Respond(payload, cmix.GetDefaultCMIXParams(), 30*time.Second)
	if err != nil {
		return errors.Errorf("unable to send restlike response message: %+v", err)
	}
	return nil
}
