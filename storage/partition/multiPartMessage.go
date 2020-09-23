package partition

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"sync"
	"time"
)

const currentMultiPartMessageVersion = 0
const keyMultiPartMessagePrefix = "MultiPartMessage"

type multiPartMessage struct {
	Sender       *id.ID
	MessageID    uint64
	NumParts     uint8
	PresentParts uint8
	Timestamp    time.Time
	MessageType  message.Type

	parts [][]byte
	kv    *versioned.KV
	mux   sync.Mutex
}

// loads an extant multipart message store or creates a new one and saves it if
// no one exists
func loadOrCreateMultiPartMessage(sender *id.ID, messageID uint64,
	kv *versioned.KV) *multiPartMessage {
	key := makeMultiPartMessageKey(sender, messageID)

	obj, err := kv.Get(key)
	if err != nil {
		if os.IsNotExist(err) {
			mpm := &multiPartMessage{
				Sender:       sender,
				MessageID:    messageID,
				NumParts:     0,
				PresentParts: 0,
				Timestamp:    time.Time{},
				MessageType:  0,
				kv:           kv,
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
	key := makeMultiPartMessageKey(mpm.Sender, mpm.MessageID)

	data, err := json.Marshal(mpm)
	if err != nil {
		return errors.Wrap(err, "Failed to unmarshal multi-part message")
	}

	obj := versioned.Object{
		Version:   currentMultiPartMessageVersion,
		Timestamp: time.Now(),
		Data:      data,
	}

	return mpm.kv.Set(key, &obj)
}

func (mpm *multiPartMessage) Add(partNumber uint8, part []byte) {
	mpm.mux.Lock()
	defer mpm.mux.Unlock()
	if len(mpm.parts) < int(partNumber) {
		mpm.parts = append(mpm.parts, make([][]byte, int(partNumber)-len(mpm.parts))...)
	}

	mpm.parts[partNumber] = part
	mpm.NumParts++

	if err := savePart(mpm.kv, mpm.Sender, mpm.MessageID, partNumber, part); err != nil {
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

func (mpm *multiPartMessage) AddFirst(mt message.Type, partNumber uint8,
	numParts uint8, timestamp time.Time, part []byte) {
	mpm.mux.Lock()
	defer mpm.mux.Unlock()
	if len(mpm.parts) < int(partNumber) {
		mpm.parts = append(mpm.parts, make([][]byte, int(partNumber)-len(mpm.parts))...)
	}

	mpm.NumParts = numParts
	mpm.Timestamp = timestamp
	mpm.MessageType = mt
	mpm.parts[partNumber] = part
	mpm.PresentParts++

	if err := savePart(mpm.kv, mpm.Sender, mpm.MessageID, partNumber, part); err != nil {
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

func (mpm *multiPartMessage) IsComplete() (message.Receive, bool) {
	mpm.mux.Lock()

	if mpm.NumParts == 0 || mpm.NumParts != mpm.PresentParts {
		mpm.mux.Unlock()
		return message.Receive{}, false
	}

	//make sure the parts buffer is large enough to load all parts from disk
	if len(mpm.parts) < int(mpm.NumParts) {
		mpm.parts = append(mpm.parts, make([][]byte, int(mpm.NumParts)-len(mpm.parts))...)
	}

	var err error
	lenMsg := 0
	//load all parts from disk, deleting files from disk as we go along
	for i := uint8(0); i < mpm.NumParts; i++ {
		if mpm.parts[i] == nil {
			if mpm.parts[i], err = loadPart(mpm.kv, mpm.Sender, mpm.MessageID, i); err != nil {
				jww.FATAL.Panicf("Failed to load multi part "+
					"message part %v from %s messageID %v: %s", i, mpm.Sender,
					mpm.MessageID, err)
			}
			if err = deletePart(mpm.kv, mpm.Sender, mpm.MessageID, i); err != nil {
				jww.FATAL.Panicf("Failed to delete  multi part "+
					"message part %v from %s messageID %v: %s", i, mpm.Sender,
					mpm.MessageID, err)
			}
		}
		lenMsg += len(mpm.parts[i])
	}

	//delete the multipart message
	mpm.delete()
	mpm.mux.Unlock()

	//reconstruct the message
	partOffset := 0
	reconstructed := make([]byte, lenMsg)
	for _, part := range mpm.parts {
		copy(reconstructed[partOffset:partOffset+len(part)], part)
		partOffset += len(part)
	}

	//return the message
	m := message.Receive{
		Payload:     reconstructed,
		MessageType: mpm.MessageType,
		Sender:      mpm.Sender,
		Timestamp:   time.Time{},
		//encryption will be set externally
		Encryption: 0,
	}

	return m, true
}

func (mpm *multiPartMessage) delete() {
	key := makeMultiPartMessageKey(mpm.Sender, mpm.MessageID)
	if err := mpm.kv.Delete(key); err != nil {
		jww.FATAL.Panicf("Failed to delete multi part "+
			"message from %s messageID %v: %s", mpm.Sender,
			mpm.MessageID, err)
	}
}

func makeMultiPartMessageKey(partner *id.ID, messageID uint64) string {
	return keyMultiPartMessagePrefix + ":" + partner.String() + ":" +
		string(messageID)
}
