///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package parse

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/parse/conversation"
	"gitlab.com/elixxir/client/e2e/parse/partition"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/storage/versioned"
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
	conversation      *conversation.Store
	partition         *partition.Store
}

func NewPartitioner(kv *versioned.KV, messageSize int) Partitioner {
	p := Partitioner{
		baseMessageSize:   messageSize,
		firstContentsSize: messageSize - firstHeaderLen,
		partContentsSize:  messageSize - headerLen,
		deltaFirstPart:    firstHeaderLen - headerLen,
		conversation:      conversation.NewStore(kv),
		partition:         partition.NewOrLoad(kv),
	}
	p.maxSize = p.firstContentsSize + (MaxMessageParts-1)*p.partContentsSize

	return p
}

func (p Partitioner) Partition(recipient *id.ID, mt catalog.MessageType,
	timestamp time.Time, payload []byte) ([][]byte, uint64, error) {

	if len(payload) > p.maxSize {
		return nil, 0, errors.Errorf("Payload is too long, max payload "+
			"length is %d, received %d", p.maxSize, len(payload))
	}

	// Get the ID of the sent message
	fullMessageID, messageID := p.conversation.Get(recipient).GetNextSendID()

	// Get the number of parts of the message; this equates to just a linear
	// equation
	numParts := uint8((len(payload) + p.deltaFirstPart + p.partContentsSize - 1) / p.partContentsSize)
	parts := make([][]byte, numParts)

	// Create the first message part
	var sub []byte
	sub, payload = splitPayload(payload, p.firstContentsSize)
	parts[0] = newFirstMessagePart(mt, messageID, numParts, timestamp, sub).bytes()

	// Create all subsequent message parts
	for i := uint8(1); i < numParts; i++ {
		sub, payload = splitPayload(payload, p.partContentsSize)
		parts[i] = newMessagePart(messageID, i, sub).bytes()
	}

	return parts, fullMessageID, nil
}

func (p Partitioner) HandlePartition(sender *id.ID,
	contents []byte, relationshipFingerprint []byte) (receive.Message, bool) {

	if isFirst(contents) {
		// If it is the first message in a set, then handle it as so

		// Decode the message structure
		fm := firstMessagePartFromBytes(contents)

		// Handle the message ID
		messageID := p.conversation.Get(sender).
			ProcessReceivedMessageID(fm.getID())
		storageTimestamp := netTime.Now()
		return p.partition.AddFirst(sender, fm.getType(), messageID,
			fm.getPart(), fm.getNumParts(), fm.getTimestamp(), storageTimestamp,
			fm.getSizedContents(), relationshipFingerprint)
	} else {
		// If it is a subsequent message part, handle it as so
		mp := messagePartFromBytes(contents)
		messageID :=
			p.conversation.Get(sender).ProcessReceivedMessageID(mp.getID())

		return p.partition.Add(sender, messageID, mp.getPart(),
			mp.getSizedContents(), relationshipFingerprint)
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
