///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package parse

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
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
	timestamp time.Time, payload []byte) ([][]byte, uint64, error) {

	if len(payload) > p.maxSize {
		return nil, 0, errors.Errorf("Payload is too long, max payload "+
			"length is %d, received %d", p.maxSize, len(payload))
	}

	// Get the ID of the sent message
	fullMessageID, messageID := p.session.Conversations().Get(recipient).GetNextSendID()

	// Get the number of parts of the message; this equates to just a linear
	// equation
	numParts := uint8((len(payload) + p.deltaFirstPart + p.partContentsSize - 1) / p.partContentsSize)
	parts := make([][]byte, numParts)

	// Create the first message part
	var sub []byte
	sub, payload = splitPayload(payload, p.firstContentsSize)
	parts[0] = newFirstMessagePart(mt, messageID, numParts, timestamp, sub).Bytes()

	// Create all subsequent message parts
	for i := uint8(1); i < numParts; i++ {
		sub, payload = splitPayload(payload, p.partContentsSize)
		parts[i] = newMessagePart(messageID, i, sub).Bytes()
	}

	return parts, fullMessageID, nil
}

func (p Partitioner) HandlePartition(sender *id.ID, _ message.EncryptionType,
	contents []byte, relationshipFingerprint []byte) (message.Receive, bool) {

	if isFirst(contents) {
		// If it is the first message in a set, then handle it as so

		// Decode the message structure
		fm := FirstMessagePartFromBytes(contents)

		// Handle the message ID
		messageID := p.session.Conversations().Get(sender).
			ProcessReceivedMessageID(fm.GetID())
		storeageTimestamp := netTime.Now()
		return p.session.Partition().AddFirst(sender, fm.GetType(),
			messageID, fm.GetPart(), fm.GetNumParts(), fm.GetTimestamp(), storeageTimestamp,
			fm.GetSizedContents(), relationshipFingerprint)
	} else {
		// If it is a subsequent message part, handle it as so
		mp := messagePartFromBytes(contents)
		messageID := p.session.Conversations().Get(sender).
			ProcessReceivedMessageID(mp.GetID())

		return p.session.Partition().Add(sender, messageID, mp.GetPart(),
			mp.GetSizedContents(), relationshipFingerprint)
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
