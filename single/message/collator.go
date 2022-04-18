package message

import (
	"bytes"
	"github.com/pkg/errors"
	"sync"
)

// Initial value of the Collator maxNum that indicates it has yet to be set.
const unsetCollatorMax = -1

// Collator stores the list of payloads in the correct order.
type Collator struct {
	payloads [][]byte // List of payloads, in order
	maxNum   int      // Max number of messages that can be received
	count    int      // Current number of messages received
	sync.Mutex
}

// NewCollator generates an empty list of payloads to fit the max number of
// possible messages. maxNum is set to indicate that it is not yet set.
func NewCollator(messageCount uint8) *Collator {
	return &Collator{
		payloads: make([][]byte, messageCount),
		maxNum:   unsetCollatorMax,
	}
}

// Collate collects message payload parts. Once all parts are received, the full
// collated payload is returned along with true. Otherwise, returns false.
func (c *Collator) Collate(payloadBytes []byte) ([]byte, bool, error) {
	payload, err := UnmarshalResponse(payloadBytes)
	if err != nil {
		return nil, false, errors.Errorf("failed to unmarshal response "+
			"payload: %+v", err)
	}

	c.Lock()
	defer c.Unlock()

	// If this is the first message received, then set the max number of
	// messages expected to be received off its max number of parts
	if c.maxNum == unsetCollatorMax {
		if int(payload.GetNumParts()) > len(c.payloads) {
			return nil, false, errors.Errorf("Max number of parts reported by "+
				"payload %d is larger than Collator expected (%d).",
				payload.GetNumParts(), len(c.payloads))
		}
		c.maxNum = int(payload.GetNumParts())
	}

	// Make sure that the part number is within the expected number of parts
	if int(payload.GetPartNum()) >= c.maxNum {
		return nil, false, errors.Errorf("Payload part number (%d) greater "+
			"than max number of expected parts (%d).",
			payload.GetPartNum(), c.maxNum)
	}

	// Make sure no payload with the same part number exists
	if c.payloads[payload.GetPartNum()] != nil {
		return nil, false, errors.Errorf("A payload for the part number %d "+
			"already exists in the list.", payload.GetPartNum())
	}

	// Add the payload to the list
	c.payloads[payload.GetPartNum()] = payload.GetContents()
	c.count++

	// Return false if not all messages have been received
	if c.count < c.maxNum {
		return nil, false, nil
	}

	// Collate all the messages
	return bytes.Join(c.payloads, nil), true, nil
}
