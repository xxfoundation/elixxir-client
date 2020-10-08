package bindings

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
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
		return -1, err
	}

	msg, err := c.api.NewCMIXMessage(u, contents)
	if err != nil {
		return -1, err
	}

	rid, err := c.api.SendCMIX(msg, params.GetDefaultCMIX())
	if err != nil {
		return -1, err
	}
	return int(rid), err
}

// SendUnsafe sends an unencrypted payload to the provided recipient
// with the provided msgType. Returns the list of rounds in which parts
// of the message were sent or an error if it fails.
// NOTE: Do not use this function unless you know what you are doing.
// This function always produces an error message in client logging.
//
// Message Types can be found in client/interfaces/message/type.go
// Make sure to not conflict with ANY default message types
func (c *Client) SendUnsafe(recipient, payload []byte,
	messageType int) (RoundList, error) {
	u, err := id.Unmarshal(recipient)
	if err != nil {
		return nil, err
	}

	m := message.Send{
		Recipient:   u,
		Payload:     payload,
		MessageType: message.Type(messageType),
	}

	rids, err := c.api.SendUnsafe(m, params.GetDefaultUnsafe())
	if err != nil {
		return nil, err
	}

	return roundList{list: rids}, nil
}
