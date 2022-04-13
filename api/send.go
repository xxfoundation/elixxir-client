///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/interfaces/params"
	e2eCrypto "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

//This holds all functions to send messages over the network

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (c *Client) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, param e2e.Params) ([]id.Round,
	e2eCrypto.MessageID, time.Time, error) {
	jww.INFO.Printf("SendE2E(%s, %d. %v)", recipient,
		mt, payload)
	return c.e2e.SendE2E(mt, recipient, payload, param)
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (c *Client) SendUnsafe(mt catalog.MessageType, recipient *id.ID,
	payload []byte, param e2e.Params) ([]id.Round, time.Time,
	error) {
	jww.INFO.Printf("SendUnsafe(%s, %d. %v)", recipient,
		mt, payload)
	return c.e2e.SendUnsafe(mt, recipient, payload, param)
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (c *Client) SendCMIX(msg format.Message, recipientID *id.ID,
	param cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	jww.INFO.Printf("Send(%s)", string(msg.GetContents()))
	return c.network.Send(recipientID, msg.GetKeyFP(),
		message.GetDefaultService(recipientID),
		msg.GetContents(), msg.GetMac(), param)
}

// SendManyCMIX sends many "raw" CMIX message payloads to each of the
// provided recipients. Used for group chat functionality. Returns the
// round ID of the round the payload was sent or an error if it fails.
func (c *Client) SendManyCMIX(messages []message.TargetedCmixMessage,
	params params.CMIX) (id.Round, []ephemeral.Id, error) {
	return c.network.SendManyCMIX(messages, params)
}

// NewCMIXMessage Creates a new cMix message with the right properties
// for the current cMix network.
// FIXME: this is weird and shouldn't be necessary, but it is.
func (c *Client) NewCMIXMessage(contents []byte) (format.Message, error) {
	primeSize := len(c.storage.Cmix().GetGroup().GetPBytes())
	msg := format.NewMessage(primeSize)
	if len(contents) > msg.ContentsSize() {
		return format.Message{}, errors.New("Contents to long for cmix")
	}
	msg.SetContents(contents)
	return msg, nil
}
