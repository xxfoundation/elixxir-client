package api

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

//This holds all functions to send messages over the network

// SendE2E sends an end-to-end payload to the provided recipient with
// the provided msgType. Returns the list of rounds in which parts of
// the message were sent or an error if it fails.
func (c *Client) SendE2E(m message.Send, param params.E2E) ([]id.Round, error) {
	jww.INFO.Printf("SendE2E(%s, %d. %v)", m.Recipient,
		m.MessageType, m.Payload)
	return c.network.SendE2E(m, param)
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
func (c *Client) SendCMIX(msg format.Message, param params.CMIX) (id.Round,
	error) {
	jww.INFO.Printf("SendCMIX(%s)", string(msg.GetContents()))
	return c.network.SendCMIX(msg, param)
}

// NewCMIXMessage Creates a new cMix message with the right properties
// for the current cMix network.
// FIXME: this is weird and shouldn't be necessary, but it is.
func (c *Client) NewCMIXMessage(recipient *id.ID,
	contents []byte) format.Message {
	primeSize := len(c.storage.Cmix().GetGroup().GetPBytes())
	msg := format.NewMessage(primeSize)
	msg.SetContents(contents)
	msg.SetRecipientID(recipient)
	return msg
}
