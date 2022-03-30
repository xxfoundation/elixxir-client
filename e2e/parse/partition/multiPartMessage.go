///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"time"
)

const currentMultiPartMessageVersion = 0
const messageKey = "MultiPart"

type multiPartMessage struct {
	Sender       *id.ID
	MessageID    uint64
	NumParts     uint8
	PresentParts uint8
	// Timestamp of message from sender
	SenderTimestamp time.Time
	// Timestamp in which message was stored in RAM
	StorageTimestamp time.Time
	MessageType      catalog.MessageType

	parts [][]byte
	kv    *versioned.KV
	mux   sync.Mutex
}

// loadOrCreateMultiPartMessage loads an extant multipart message store or
// creates a new one and saves it if one does not exist.
func loadOrCreateMultiPartMessage(sender *id.ID, messageID uint64,
	kv *versioned.KV) *multiPartMessage {
	kv = kv.Prefix(versioned.MakePartnerPrefix(sender)).Prefix(fmt.Sprintf("MessageID:%d", messageID))

	obj, err := kv.Get(messageKey, currentMultiPartMessageVersion)
	if err != nil {
		if !ekv.Exists(err) {
			mpm := &multiPartMessage{
				Sender:          sender,
				MessageID:       messageID,
				NumParts:        0,
				PresentParts:    0,
				SenderTimestamp: time.Time{},
				MessageType:     0,
				kv:              kv,
			}
			if err = mpm.save(); err != nil {
				jww.FATAL.Panicf("Failed to save new multi part "+
					"message from %s messageID %v: %s", sender, messageID, err)
			}
			return mpm
		}
		jww.FATAL.Panicf("Failed to open multi part "+
			"message from %s messageID %v: %s", sender, messageID, err)
	}

	mpm := &multiPartMessage{
		kv: kv,
	}

	if err = json.Unmarshal(obj.Data, mpm); err != nil {
		jww.FATAL.Panicf("Failed to unmarshal multi part "+
			"message from %s messageID %v: %s", sender, messageID, err)
	}

	return mpm
}

func (mpm *multiPartMessage) save() error {
	data, err := json.Marshal(mpm)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal multi-part message")
	}

	obj := versioned.Object{
		Version:   currentMultiPartMessageVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return mpm.kv.Set(messageKey, currentMultiPartMessageVersion, &obj)
}

func (mpm *multiPartMessage) Add(partNumber uint8, part []byte) {
	mpm.mux.Lock()
	defer mpm.mux.Unlock()

	// Extend the list if needed
	if len(mpm.parts) <= int(partNumber) {
		mpm.parts = append(mpm.parts, make([][]byte, int(partNumber)-len(mpm.parts)+1)...)
	}

	mpm.parts[partNumber] = part
	mpm.PresentParts++

	if err := savePart(mpm.kv, partNumber, part); err != nil {
		jww.FATAL.Panicf("Failed to save multi part "+
			"message part %v from %s messageID %v: %s", partNumber, mpm.Sender,
			mpm.MessageID, err)
	}

	if err := mpm.save(); err != nil {
		jww.FATAL.Panicf("Failed to save multi part "+
			"message after adding part %v from %s messageID %v: %s", partNumber,
			mpm.Sender, mpm.MessageID, err)
	}
}

func (mpm *multiPartMessage) AddFirst(mt catalog.MessageType, partNumber uint8,
	numParts uint8, senderTimestamp, storageTimestamp time.Time, part []byte) {
	mpm.mux.Lock()
	defer mpm.mux.Unlock()

	// Extend the list if needed
	if len(mpm.parts) <= int(partNumber) {
		mpm.parts = append(mpm.parts, make([][]byte, int(partNumber)-len(mpm.parts)+1)...)
	}

	mpm.NumParts = numParts
	mpm.SenderTimestamp = senderTimestamp
	mpm.MessageType = mt
	mpm.parts[partNumber] = part
	mpm.PresentParts++
	mpm.StorageTimestamp = storageTimestamp

	if err := savePart(mpm.kv, partNumber, part); err != nil {
		jww.FATAL.Panicf("Failed to save multi part "+
			"message part %v from %s messageID %v: %s", partNumber, mpm.Sender,
			mpm.MessageID, err)
	}

	if err := mpm.save(); err != nil {
		jww.FATAL.Panicf("Failed to save multi part message after adding part "+
			"%v from %s messageID %v: %s",
			partNumber, mpm.Sender, mpm.MessageID, err)
	}
}

func (mpm *multiPartMessage) IsComplete(relationshipFingerprint []byte) (receive.Message, bool) {
	mpm.mux.Lock()
	if mpm.NumParts == 0 || mpm.NumParts != mpm.PresentParts {
		mpm.mux.Unlock()
		return receive.Message{}, false
	}

	// Make sure the parts buffer is large enough to load all parts from disk
	if len(mpm.parts) < int(mpm.NumParts) {
		mpm.parts = append(mpm.parts, make([][]byte, int(mpm.NumParts)-len(mpm.parts))...)
	}

	// delete the multipart message
	lenMsg := mpm.delete()
	mpm.mux.Unlock()

	// Reconstruct the message
	partOffset := 0
	reconstructed := make([]byte, lenMsg)
	for _, part := range mpm.parts {
		copy(reconstructed[partOffset:partOffset+len(part)], part)
		partOffset += len(part)
	}

	var mid e2e.MessageID
	if len(relationshipFingerprint) != 0 {
		mid = e2e.NewMessageID(relationshipFingerprint, mpm.MessageID)
	}

	// Return the message
	m := receive.Message{
		Payload:     reconstructed,
		MessageType: mpm.MessageType,
		Sender:      mpm.Sender,
		Timestamp:   mpm.SenderTimestamp,
		ID:          mid,
	}

	return m, true
}

// deletes all parts from disk and RAM. Returns the message length for reconstruction
func (mpm *multiPartMessage) delete() int {
	// Load all parts from disk, deleting files from disk as we go along
	var err error
	lenMsg := 0
	for i := uint8(0); i < mpm.NumParts; i++ {
		if mpm.parts[i] == nil {
			if mpm.parts[i], err = loadPart(mpm.kv, i); err != nil {
				jww.FATAL.Panicf("Failed to load multi part "+
					"message part %v from %s messageID %v: %s", i, mpm.Sender,
					mpm.MessageID, err)
			}
			if err = deletePart(mpm.kv, i); err != nil {
				jww.FATAL.Panicf("Failed to delete  multi part "+
					"message part %v from %s messageID %v: %s", i, mpm.Sender,
					mpm.MessageID, err)
			}
		}
		lenMsg += len(mpm.parts[i])
	}

	//key := makeMultiPartMessageKey(mpm.MessageID)
	if err := mpm.kv.Delete(messageKey,
		currentMultiPartMessageVersion); err != nil {
		jww.FATAL.Panicf("Failed to delete multi part "+
			"message from %s messageID %v: %s", mpm.Sender,
			mpm.MessageID, err)
	}

	return lenMsg
}
