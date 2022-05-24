///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"encoding/json"
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/primitives/id"
	"time"
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
func (c *Client) SendCmix(recipient, contents []byte, parameters string) (int, error) {
	p, err := params.GetCMIXParameters(parameters)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}

	u, err := id.Unmarshal(recipient)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}

	msg, err := c.api.NewCMIXMessage(contents)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}

	rid, _, err := c.api.SendCMIX(msg, u, p)
	if err != nil {
		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
			err))
	}
	return int(rid), nil
}

// SendManyCMIX sends many "raw" CMIX message payloads to each of the
// provided recipients. Used for group chat functionality. Returns the
// round ID of the round the payload was sent or an error if it fails.
// This will return an error if:
//  - any recipient ID is invalid
//  - any of the the message contents are too long for the message structure
//  - the message cannot be sent

// This will return the round the message was sent on if it is successfully sent
// This can be used to register a round event to learn about message delivery.
// on failure a round id of -1 is returned
// fixme: cannot use a slice of slices over bindings. Will need to modify this function once
//  a proper input format has been specified
// func (c *Client) SendManyCMIX(recipients, contents [][]byte, parameters string) (int, error) {
//
//	p, err := params.GetCMIXParameters(parameters)
//	if err != nil {
//		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
//			err))
//	}
//
//	// Build messages
//	messages := make(map[id.ID]format.Message, len(contents))
//	for i := 0; i < len(contents); i++ {
//		msg, err := c.api.NewCMIXMessage(contents[i])
//		if err != nil {
//			return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
//				err))
//		}
//
//		u, err := id.Unmarshal(recipients[i])
//		if err != nil {
//			return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
//				err))
//		}
//
//		messages[*u] = msg
//	}
//
//	rid, _, err := c.api.SendManyCMIX(messages, p)
//	if err != nil {
//		return -1, errors.New(fmt.Sprintf("Failed to sendCmix: %+v",
//			err))
//	}
//	return int(rid), nil
// }

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
//
// Message Types can be found in client/interfaces/message/type.go
// Make sure to not conflict with ANY default message types with custom types
func (c *Client) SendUnsafe(recipient, payload []byte,
	messageType int, parameters string) (*RoundList, error) {
	p, err := params.GetUnsafeParameters(parameters)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to sendUnsafe: %+v",
			err))
	}
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

	rids, err := c.api.SendUnsafe(m, p)
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
func (c *Client) SendE2E(recipient, payload []byte, messageType int, parameters string) (*SendReport, error) {
	p, err := params.GetE2EParameters(parameters)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed SendE2E: %+v", err))
	}

	u, err := id.Unmarshal(recipient)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed SendE2E: %+v", err))
	}

	m := message.Send{
		Recipient:   u,
		Payload:     payload,
		MessageType: message.Type(messageType),
	}

	rids, mid, ts, err := c.api.SendE2E(m, p)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed SendE2E: %+v", err))
	}

	sr := SendReport{
		rl:  &RoundList{list: rids},
		mid: mid,
		ts:  ts,
	}

	return &sr, nil
}

// the send report is the mechanisim by which sendE2E returns a single
type SendReport struct {
	rl  *RoundList
	mid e2e.MessageID
	ts  time.Time
}

type SendReportDisk struct {
	List []id.Round
	Mid  []byte
	Ts   int64
}

func (sr *SendReport) GetRoundList() *RoundList {
	return sr.rl
}

func (sr *SendReport) GetMessageID() []byte {
	return sr.mid[:]
}

func (sr *SendReport) GetRoundURL() string {
	if sr.rl != nil && sr.rl.Len() > 0 {
		return getRoundURL(sr.rl.list[0])
	}
	return dashboardBaseURL
}

// GetTimestampMS returns the message's timestamp in milliseconds
func (sr *SendReport) GetTimestampMS() int64 {
	ts := sr.ts.UnixNano()
	ts = (ts + 500000) / 1000000
	return ts
}

// GetTimestampNano returns the message's timestamp in nanoseconds
func (sr *SendReport) GetTimestampNano() int64 {
	return sr.ts.UnixNano()
}

func (sr *SendReport) Marshal() ([]byte, error) {
	srd := SendReportDisk{
		List: sr.rl.list,
		Mid:  sr.mid[:],
		Ts:   sr.ts.UnixNano(),
	}
	return json.Marshal(&srd)
}

func (sr *SendReport) Unmarshal(b []byte) error {
	srd := SendReportDisk{}
	if err := json.Unmarshal(b, &srd); err != nil {
		return errors.New(fmt.Sprintf("Failed to unmarshal send "+
			"report: %s", err.Error()))
	}

	copy(sr.mid[:], srd.Mid)
	sr.rl = &RoundList{list: srd.List}
	sr.ts = time.Unix(0, srd.Ts)
	return nil
}
