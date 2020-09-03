package collate

import (
	"crypto/md5"
	"encoding/binary"
	"gitlab.com/xx_network/primitives/id"
)

type multiPartID [16]byte

type Store struct {
	multiparts map[multiPartID]multiPartMessage
}

func getMultiPartID(partner *id.ID, messageID uint64) multiPartID {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, messageID)
	return md5.Sum(append(partner[:], b...))
}
