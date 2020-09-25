package parse

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const MaxMessageParts = 255

type Partitioner struct {
	baseMessageSize   int
	firstContentsSize int
	partContentsSize  int
	deltaFirstPart    int
	maxSize           int
	session           *storage.Session
}

func NewPartitioner(messageSize int, session *storage.Session) Partitioner {
	p := Partitioner{
		baseMessageSize:   messageSize,
		firstContentsSize: messageSize - firstHeaderLen,
		partContentsSize:  messageSize - headerLen,
		deltaFirstPart:    firstHeaderLen - headerLen,
		session:           session,
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

	//Get the ID of the sent message
	_, messageID := p.session.Conversations().Get(recipient).GetNextSendID()

	// get the number of parts of the message. This equates to just a linear
	// equation
	numParts := uint8((len(payload) + p.deltaFirstPart + p.partContentsSize - 1) / p.partContentsSize)
	parts := make([][]byte, numParts)

	//Create the first message part
	var sub []byte
	sub, payload = splitPayload(payload, p.firstContentsSize)
	parts[0] = newFirstMessagePart(mt, messageID, numParts, timestamp, sub).Bytes()

	//create all subsiquent message parts
	for i := uint8(1); i < numParts; i++ {
		sub, payload = splitPayload(payload, p.partContentsSize)
		parts[i] = newMessagePart(messageID, i, sub).Bytes()
	}

	return parts, nil
}

func (p Partitioner) HandlePartition(sender *id.ID, e message.EncryptionType,
	contents []byte) (message.Receive, bool) {

	//If it is the first message in a set, handle it as so
	if isFirst(contents) {
		//decode the message structure
		fm := FirstMessagePartFromBytes(contents)
		timestamp, err := fm.GetTimestamp()
		if err != nil {
			jww.FATAL.Panicf("Failed Handle Partition, failed to get "+
				"timestamp message from %s messageID %v: %s", sender,
				fm.Timestamp, err)
		}

		//Handle the message ID
		messageID := p.session.Conversations().Get(sender).
			ProcessReceivedMessageID(fm.GetID())

		//Return the
		return p.session.Partition().AddFirst(sender, fm.GetType(),
			messageID, fm.GetPart(), fm.GetNumParts(), timestamp,
			fm.GetSizedContents())
		//If it is a subsiquent message part, handle it as so
	} else {
		mp := MessagePartFromBytes(contents)
		messageID := p.session.Conversations().Get(sender).
			ProcessReceivedMessageID(mp.GetID())

		return p.session.Partition().Add(sender, messageID, mp.GetPart(),
			mp.GetSizedContents())
	}
}

func splitPayload(payload []byte, length int) ([]byte, []byte) {
	if len(payload) < length {
		return payload, payload
	}
	return payload[:length], payload[length:]
}

func isFirst(payload []byte) bool {
	return payload[idLen] == 0
}
