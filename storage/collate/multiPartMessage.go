package collate

import (
	"gitlab.com/elixxir/client/context/message"
	"time"
)

type multiPartMessage struct {
	messageID    uint64
	numParts     uint8
	presentParts uint8
	timestamp    time.Time
	messageType  message.Type

	parts [][]byte
}
