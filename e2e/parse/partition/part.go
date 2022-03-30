///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package partition

import (
	"fmt"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
)

const currentMultiPartMessagePartVersion = 0

func loadPart(kv *versioned.KV, partNum uint8) ([]byte, error) {
	key := makeMultiPartMessagePartKey(partNum)

	obj, err := kv.Get(key, currentMultiPartMessageVersion)
	if err != nil {
		return nil, err
	}

	return obj.Data, nil
}

func savePart(kv *versioned.KV, partNum uint8, part []byte) error {
	key := makeMultiPartMessagePartKey(partNum)

	obj := versioned.Object{
		Version:   currentMultiPartMessagePartVersion,
		Timestamp: netTime.Now(),
		Data:      part,
	}

	return kv.Set(key, currentMultiPartMessageVersion, &obj)
}

func deletePart(kv *versioned.KV, partNum uint8) error {
	key := makeMultiPartMessagePartKey(partNum)
	return kv.Delete(key, currentMultiPartMessageVersion)
}

// Make the key for a part
func makeMultiPartMessagePartKey(part uint8) string {
	return fmt.Sprintf("part:%v", part)
}

//func multiPartMessagePartPrefix(kv *versioned.KV, id uint64) *versioned.KV {
//	return kv.prefix(keyMultiPartMessagePartPrefix).
//		prefix(strconv.FormatUint(id, 32))
//}
