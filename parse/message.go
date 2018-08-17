////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"crypto/sha256"
	"gitlab.com/privategrity/client/user"
)

const MessageHashLenBits = uint64(256)
const MessageHashLen = MessageHashLenBits / 8

type MessageHash [MessageHashLen]byte

type Message struct {
	TypedBody
	Sender   user.ID
	Receiver user.ID
	Nonce    []byte
}

func (m Message) Hash() MessageHash {
	var mh MessageHash

	h := sha256.New()

	h.Write(m.Type.Bytes())
	h.Write(m.Body)
	h.Write(m.Sender.Bytes())
	h.Write(m.Receiver.Bytes())
	//h.Write(m.Nonce)

	hashed := h.Sum(nil)

	copy(mh[:], hashed[:MessageHashLen])

	return mh
}

func (m Message) GetSender() []byte {
	return m.Sender.Bytes()
}

func (m Message) GetRecipient() []byte {
	return m.Receiver.Bytes()
}

func (m Message) GetPayload() string {
	return string(Pack(&m.TypedBody))
}
