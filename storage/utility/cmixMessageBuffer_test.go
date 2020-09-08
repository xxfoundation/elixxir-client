package utility

import (
	"bytes"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"math/rand"
	"testing"
	"time"
)

func Test_saveMessage(t *testing.T) {
	// Set up test values
	cmh := &cmixMessageHandler{}

	kv := versioned.NewKV(make(ekv.Memstore))
	subKey := "testKey"
	testMsgs, _ := makeTestCmixMessages(1)
	mh := cmh.HashMessage(testMsgs[0])
	key := makeStoredMessageKey(subKey, mh)

	// Save message
	err := cmh.SaveMessage(kv, testMsgs[0], key)
	if err != nil {
		t.Errorf("saveMessage() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	// Try to get message
	obj, err := kv.Get(key)
	if err != nil {
		t.Errorf("Get() returned an error."+
			"\n\texpected: %v\n\trecieved: %v", nil, err)
	}

	if !bytes.Equal(testMsgs[0].Marshal(), obj.Data) {
		t.Errorf("saveMessage() returned versioned object with incorrect data."+
			"\n\texpected: %v\n\treceived: %v",
			testMsgs[0], obj.Data)
	}
}

// makeTestCmixMessages creates a list of messages with random data and the expected
// map after they are added to the buffer.
// makeTestMessages creates a list of messages with random data and the expected
// map after they are added to the buffer.
func makeTestCmixMessages(n int) ([]format.Message, map[MessageHash]struct{}) {
	cmh := &cmixMessageHandler{}
	prng := rand.New(rand.NewSource(time.Now().UnixNano()))
	mh := map[MessageHash]struct{}{}
	msgs := make([]format.Message, n)
	for i := range msgs {
		msgs[i] = format.NewMessage(128)
		payload := make([]byte, 128)
		prng.Read(payload)
		msgs[i].SetPayloadA(payload)
		prng.Read(payload)
		msgs[i].SetPayloadB(payload)
		mh[cmh.HashMessage(msgs[i])] = struct{}{}
	}

	return msgs, mh
}
