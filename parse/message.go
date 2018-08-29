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

const MessageHashLenBits = 256
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

	h.Write(TypeAsBytes(int32(m.Type)))
	h.Write(m.Body)
	h.Write([]byte(m.Sender))
	h.Write([]byte(m.Receiver))
	//h.Write(m.Nonce)

	hashed := h.Sum(nil)

	copy(mh[:], hashed[:MessageHashLen])

	return mh
}

func (m Message) GetSender() []byte {
	return []byte(m.Sender)
}

func (m Message) GetRecipient() []byte {
	return []byte(m.Receiver)
}

func (m Message) GetPayload() string {
	return string(Pack(&m.TypedBody))
}
