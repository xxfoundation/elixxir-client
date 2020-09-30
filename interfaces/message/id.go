package message

import (
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/primitives/id"
)

const messageIDLen = 32

type ID [messageIDLen]byte

func NewID(sender, receiver *id.ID, connectionSalt []byte,
	internalMessageID uint64) ID {
	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("Failed to get hash for message ID creation")
	}

	h.Write(sender.Bytes())
	h.Write(receiver.Bytes())

	intMidBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(intMidBytes, internalMessageID)

	h.Write(intMidBytes)
	h.Write(connectionSalt)

	midBytes := h.Sum(nil)

	mid := ID{}
	copy(mid[:], midBytes)
	return mid
}
