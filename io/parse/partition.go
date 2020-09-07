package parse

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/xx_network/primitives/id"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

const MaxMessageParts = 255

type Partitioner struct {
	baseMessageSize   int
	firstContentsSize int
	partContentsSize  int
	deltaFirstPart    int
	maxSize           int
	ctx               context.Context
}

func NewPartitioner(messageSize int, ctx context.Context) Partitioner {
	p := Partitioner{
		baseMessageSize:   messageSize,
		firstContentsSize: messageSize - firstHeaderLen,
		partContentsSize:  messageSize - headerLen,
		deltaFirstPart:    firstHeaderLen - headerLen,
		ctx:               ctx,
	}
	p.maxSize = p.firstContentsSize + (MaxMessageParts-1)*p.partContentsSize
	return p
}

func (p Partitioner) Partition(recipient *id.ID, mt message.Type,
	timestamp time.Time, payload []byte) ([][]byte, error) {

	if len(payload) > p.maxSize {
		return nil, errors.Errorf("Payload is too long, max payload "+
			"length is %v, received %v", p.maxSize, len(payload))
	}

	_, messageID := p.ctx.Session.Conversations().Get(recipient).GetNextSendID()

	numParts := uint8((len(payload) + p.deltaFirstPart + p.partContentsSize - 1) / p.partContentsSize)
	parts := make([][]byte, numParts)

	var sub []byte
	sub, payload = splitPayload(payload, p.firstContentsSize)
	parts[0] = newFirstMessagePart(mt, messageID, 0, numParts, timestamp, sub).Bytes()

	for i := uint8(1); i < numParts; i++ {
		sub, payload = splitPayload(payload, p.partContentsSize)
		parts[i] = newMessagePart(messageID, i, sub).Bytes()
	}

	return parts, nil
}

func (p Partitioner) HandlePartition(sender *id.ID, e message.EncryptionType,
	contents []byte) (message.Receive, bool, error) {
	//if it is a raw message, there is nothing to do
	if isRaw(contents) {
		return message.Receive{
			Payload:     contents,
			MessageType: message.Raw,
			Sender:      sender,
			Timestamp:   time.Time{},
			Encryption:  e,
		}, true, nil
	}

	if isFirst(contents) {
		fm := FirstMessagePartFromBytes(contents)
		timestamp, err := fm.GetTimestamp()
		if err != nil {
			jww.FATAL.Panicf("Failed Handle Partition, failed to get "+
				"timestamp message from %s messageID %v: %s", sender,
				fm.Timestamp, err)
		}

		messageID := p.ctx.Session.Conversations().Get(sender).
			ProcessReceivedMessageID(fm.GetID())

		m, ok := p.ctx.Session.Partition().AddFirst(sender, fm.GetType(),
			messageID, fm.GetPart(), fm.GetNumParts(), timestamp,
			fm.GetContents())
		if ok {
			return m, true, nil
		} else {
			return message.Receive{}, false, nil
		}
	} else {
		mp := MessagePartFromBytes(contents)
		messageID := p.ctx.Session.Conversations().Get(sender).
			ProcessReceivedMessageID(mp.GetID())

		m, ok := p.ctx.Session.Partition().Add(sender, messageID, mp.GetPart(),
			mp.GetContents())
		if ok {
			return m, true, nil
		} else {
			return message.Receive{}, false, nil
		}
	}
}

func splitPayload(payload []byte, length int) ([]byte, []byte) {
	if len(payload) < length {
		return payload, payload
	}
	return payload[:length], payload[length:]
}

func isRaw(payload []byte) bool {
	return payload[0]&0b10000000 == 0
}

func isFirst(payload []byte) bool {
	return payload[idLen] == 0
}
