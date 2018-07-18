////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"gitlab.com/privategrity/client/user"
	"crypto/sha256"
	"fmt"
)

const MessageHashLenBits = uint64(256)
const MessageHashLen = MessageHashLenBits/8

type MessageHash [MessageHashLen]byte

type Message struct {
	TypedBody
	Sender   user.ID
	Receiver user.ID
	Nonce 	 []byte
}

func (m Message) Hash()MessageHash{
	var mh MessageHash

	h := sha256.New()

	h.Write(m.Type.Bytes())
	h.Write(m.Body)
	h.Write(m.Sender.Bytes())
	h.Write(m.Receiver.Bytes())
	h.Write(m.Nonce)

	hashed := h.Sum(nil)

	fmt.Println(len(hashed))

	copy(mh[:],hashed[:MessageHashLen])

	return mh
}
