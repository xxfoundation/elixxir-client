////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"crypto/sha256"
	"gitlab.com/privategrity/crypto/id"
	"gitlab.com/privategrity/client/cmixproto"
)

const MessageHashLenBits = 256
const MessageHashLen = MessageHashLenBits / 8

type MessageHash [MessageHashLen]byte

type Message struct {
	TypedBody
	Sender   *id.UserID
	Receiver *id.UserID
	Nonce    []byte
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

func (m *Message) GetSender() *id.UserID {
	return m.Sender
}

func (m *Message) GetRecipient() *id.UserID {
	return m.Receiver
}

func (m *Message) GetPayload() []byte {
	return Pack(&m.TypedBody)
}

func (m *Message) GetType() cmixproto.Type {
	return m.Type
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
// It's essentially a binary blob.
func (p *BindingsMessageProxy) GetPayload() []byte {
	return p.Proxy.GetPayload()
}

func (p *BindingsMessageProxy) GetType() int32 {
	return int32(p.Proxy.GetType())
}
