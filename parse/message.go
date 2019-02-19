////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"crypto/sha256"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/primitives/id"
)

const MessageHashLenBits = 256
const MessageHashLen = MessageHashLenBits / 8

type MessageHash [MessageHashLen]byte

type Message struct {
	TypedBody
	Sender   *id.User
	Receiver *id.User
	Nonce    []byte
}

// Interface used to standardize message definitions
type MessageInterface interface {
	// Returns the message's sender ID
	// (uint64) BigEndian serialized into a byte slice
	GetSender() *id.User
	// Returns the message payload, without packed type
	GetPayload() []byte
	// Returns the message's recipient ID
	// (uint64) BigEndian serialized into a byte slice
	GetRecipient() *id.User
	// Return the message's type
	GetType() cmixproto.Type
	// Return the message fully serialized including the type prefix
	// Does this really belong in the interface?
	Pack() []byte
}

func (m Message) Hash() MessageHash {
	var mh MessageHash

	h := sha256.New()

	h.Write(TypeAsBytes(int32(m.Type)))
	h.Write(m.Body)
	h.Write(m.Sender.Bytes())
	h.Write(m.Receiver.Bytes())
	//h.Write(m.Nonce)

	hashed := h.Sum(nil)

	copy(mh[:], hashed[:MessageHashLen])

	return mh
}

func (m *Message) GetSender() *id.User {
	return m.Sender
}

func (m *Message) GetRecipient() *id.User {
	return m.Receiver
}

func (m *Message) GetPayload() []byte {
	return m.Body
}

func (m *Message) GetType() cmixproto.Type {
	return m.Type
}

func (m *Message) Pack() []byte {
	return Pack(&m.TypedBody)
}

// Implements Message type compatibility with bindings package's limited types
type BindingsMessageProxy struct {
	Proxy *Message
}

func (p *BindingsMessageProxy) GetSender() []byte {
	return p.Proxy.GetSender().Bytes()
}

func (p *BindingsMessageProxy) GetRecipient() []byte {
	return p.Proxy.GetRecipient().Bytes()
}

// TODO Should we actually pass this over the boundary as a byte slice?
// It's essentially a binary blob, so probably yes.
func (p *BindingsMessageProxy) GetPayload() []byte {
	return p.Proxy.GetPayload()
}

func (p *BindingsMessageProxy) GetType() int32 {
	return int32(p.Proxy.GetType())
}

// Includes the type. Not sure if this is the right way to approach this.
func (p *BindingsMessageProxy) Pack() []byte {
	return Pack(&TypedBody{
		Type: p.Proxy.GetType(),
		Body: p.Proxy.GetPayload(),
	})
}
