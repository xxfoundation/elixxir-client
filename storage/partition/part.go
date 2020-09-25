package partition

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"time"
)

const currentMultiPartMessagePartVersion = 0
const keyMultiPartMessagePartPrefix = "MultiPartMessagePart"

func loadPart(kv *versioned.KV, partner *id.ID, messageID uint64, partNum uint8) ([]byte, error) {
	key := makeMultiPartMessagePartKey(partner, messageID, partNum)

	obj, err := kv.Get(key)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

func savePart(kv *versioned.KV, partner *id.ID, messageID uint64, partNum uint8, part []byte) error {
	key := makeMultiPartMessagePartKey(partner, messageID, partNum)

	obj := versioned.Object{
		Version:   currentMultiPartMessagePartVersion,
		Timestamp: time.Now(),
		Data:      part,
	}

	return kv.Set(key, &obj)
}

func deletePart(kv *versioned.KV, partner *id.ID, messageID uint64, partNum uint8) error {
	key := makeMultiPartMessagePartKey(partner, messageID, partNum)
	return kv.Delete(key)
}

func makeMultiPartMessagePartKey(partner *id.ID, messageID uint64, partNum uint8) string {
	return keyMultiPartMessagePartPrefix + ":" + partner.String() + ":" +
		strconv.FormatUint(messageID, 10) + ":" + string(partNum)

}
