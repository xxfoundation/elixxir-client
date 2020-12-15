///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
)

// SendCMIX sends a "raw" CMIX message payload to the provided
// recipient. Note that both SendE2E and SendUnsafe call SendCMIX.
// Returns the round ID of the round the payload was sent or an error
// if it fails.

// This will return an error if:
//  - the recipient ID is invalid
//  - the contents are too long for the message structure
//  - the message cannot be sent

// This will return the round the message was sent on if it is successfully sent
// This can be used to register a round event to learn about message delivery.
// on failure a round id of -1 is returned
func (c *Client) SendCmix(recipient, contents []byte) (int, error) {
	u, err := id.Unmarshal(recipient)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}

	msg, err := c.api.NewCMIXMessage(u, contents)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}

	rid, err := c.api.SendCMIX(msg, params.GetDefaultCMIX())
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}
	return int(rid), nil
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
//
// Message Types can be found in client/interfaces/message/type.go
// Make sure to not conflict with ANY default message types with custom types
func (c *Client) SendUnsafe(recipient, payload []byte,
	messageType int) (*RoundList, error) {
	u, err := id.Unmarshal(recipient)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to sendUnsafe: %+v",
			err))
	}

	m := message.Send{
		Recipient:   u,
		Payload:     payload,
		MessageType: message.Type(messageType),
	}

	rids, err := c.api.SendUnsafe(m, params.GetDefaultUnsafe())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to sendUnsafe: %+v",
			err))
	}

	return &RoundList{list: rids}, nil
}

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
//
// Message Types can be found in client/interfaces/message/type.go
// Make sure to not conflict with ANY default message types
func (c *Client) SendE2E(recipient, payload []byte, messageType int) (*SendReport, error) {
	u, err := id.Unmarshal(recipient)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed SendE2E: %+v", err))
	}

	m := message.Send{
		Recipient:   u,
		Payload:     payload,
		MessageType: message.Type(messageType),
	}

	rids, mid, err := c.api.SendE2E(m, params.GetDefaultE2E())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed SendE2E: %+v", err))
	}

	sr := SendReport{
		rl:  &RoundList{list: rids},
		mid: mid,
	}

	return &sr, nil
}

// the send report is the mechanisim by which sendE2E returns a single
type SendReport struct {
	rl  *RoundList
	mid e2e.MessageID
}

func (sr *SendReport) GetRoundList() *RoundList {
	return sr.rl
}

func (sr *SendReport) GetMessageID() []byte {
	return sr.mid[:]
}

func (sr *SendReport) Marshal() ([]byte, error) {
	return json.Marshal(sr)
}
