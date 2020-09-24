package partition

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"strconv"
	"time"
)

const currentMultiPartMessagePartVersion = 0
const keyMultiPartMessagePartPrefix = "parts"

func loadPart(kv *versioned.KV, messageID uint64, partNum uint8) ([]byte, error) {
	kv = kv.Prefix(keyMultiPartMessagePartPrefix).
		Prefix(strconv.FormatUint(messageID, 32))
	key := makeMultiPartMessagePartKey(partNum)

	obj, err := kv.Get(key)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

func savePart(kv *versioned.KV, messageID uint64, partNum uint8, part []byte) error {
	kv = kv.Prefix(keyMultiPartMessagePartPrefix).
		Prefix(strconv.FormatUint(messageID, 32))
	key := makeMultiPartMessagePartKey(partNum)

	obj := versioned.Object{
		Version:   currentMultiPartMessagePartVersion,
		Timestamp: time.Now(),
		Data:      part,
	}

	return kv.Set(key, &obj)
}

func deletePart(kv *versioned.KV, messageID uint64, partNum uint8) error {
	kv = kv.Prefix(keyMultiPartMessagePartPrefix).
		Prefix(strconv.FormatUint(messageID, 32))
	key := makeMultiPartMessagePartKey(partNum)
	return kv.Delete(key)
}

func makeMultiPartMessagePartKey(partNum uint8) string {
	return strconv.FormatUint(uint64(partNum), 32)
}
