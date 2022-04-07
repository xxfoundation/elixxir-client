///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

//This holds all functions to send messages over the network

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (c *Client) SendE2E(m message.Send, param params.E2E) ([]id.Round,
	e2e.MessageID, time.Time, error) {
	jww.INFO.Printf("SendE2E(%s, %d. %v)", m.Recipient,
		m.MessageType, m.Payload)
	return c.network.SendE2E(m, param, nil)
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
func (c *Client) SendUnsafe(m message.Send, param params.Unsafe) ([]id.Round,
	error) {
	jww.INFO.Printf("SendUnsafe(%s, %d. %v)", m.Recipient,
		m.MessageType, m.Payload)
	return c.network.SendUnsafe(m, param)
}

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.
func (c *Client) SendCMIX(msg format.Message, recipientID *id.ID,
	param params.CMIX) (id.Round, ephemeral.Id, error) {
	jww.INFO.Printf("Send(%s)", string(msg.GetContents()))
	return c.network.SendCMIX(msg, recipientID, param)
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
