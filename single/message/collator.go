////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"bytes"
	"github.com/pkg/errors"
	"sync"
)

// Error messages.
const (
	// Collate
	errMaxParts       = "max number of parts reported by payload %d is larger than collator expected (%d)"
	errPartOutOfRange = "payload part number %d greater than max number of expected parts (%d)"
	errPartExists     = "a payload for the part number %d already exists in the list"
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

type Part interface {
	// GetNumParts returns the total number of parts in the message.
	GetNumParts() uint8

	// GetPartNum returns the index of this part in the message.
	GetPartNum() uint8

	// GetContents returns the contents of the message part.
	GetContents() []byte
}

// NewCollator generates an empty list of payloads to fit the max number of
// possible messages. maxNum is set to indicate that it is not yet set.
func NewCollator(messageCount uint8) *Collator {
	return &Collator{
		payloads: make([][]byte, messageCount),
		maxNum:   unsetCollatorMax,
		count:    0,
	}
}

// Collate collects message payload parts. Once all parts are received, the full
// collated payload is returned along with true. Otherwise, returns false.
func (c *Collator) Collate(part Part) ([]byte, bool, error) {
	c.Lock()
	defer c.Unlock()

	// If this is the first message received, then set the max number of
	// messages expected to be received off its max number of parts
	if c.maxNum == unsetCollatorMax {
		if int(part.GetNumParts()) > len(c.payloads) {
			return nil, false, errors.Errorf(
				errMaxParts, part.GetNumParts(), len(c.payloads))
		}
		c.maxNum = int(part.GetNumParts())
	}

	// Make sure that the part number is within the expected number of parts
	if int(part.GetPartNum()) >= c.maxNum {
		return nil, false,
			errors.Errorf(errPartOutOfRange, part.GetPartNum(), c.maxNum)
	}

	// Make sure no payload with the same part number exists
	if c.payloads[part.GetPartNum()] != nil {
		return nil, false, errors.Errorf(errPartExists, part.GetPartNum())
	}

	// Add the payload to the list
	c.payloads[part.GetPartNum()] = part.GetContents()
	c.count++

	// Return false if not all messages have been received
	if c.count < c.maxNum {
		return nil, false, nil
	}

	// Collate all the messages
	return bytes.Join(c.payloads, nil), true, nil
}
