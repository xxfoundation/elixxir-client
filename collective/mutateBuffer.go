package collective

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/ekv"
	"strconv"
	"sync/atomic"
)

const (
	bufferMarker   = "⌛"
	buffElementKey = "⌛bufferElement_"
	bufferSize     = 1024
)

type mutateBuffer struct {
	index *uint64
	kv    ekv.KeyValue
}

func loadBuffer(kv ekv.KeyValue) (*mutateBuffer, map[int]map[string]Mutate) {
	transactions := make(map[int]map[string]Mutate, bufferSize)

	var largestBucketSize, largestBucketStart,
		currentBucketSize, currentBucketStart int

	firstBucketStart := bufferSize

	for i := 0; i < bufferSize; i++ {
		elementKey := createElementKey(i)
		data, err := kv.GetBytes(elementKey)
		if err != nil {
			if ekv.Exists(err) {
				jww.WARN.Printf("failed to load from mutate buffer at "+
					"index %d(%s): %+v", i, elementKey, err)
			}
			currentBucketSize = currentBucketSize + 1
			continue
		}

		var mutates map[string]Mutate
		if err = json.Unmarshal(data, &mutates); err != nil {
			if ekv.Exists(err) {
				jww.WARN.Printf("unmarshal to load from mutate buffer at "+
					"index %d(%s): %+v", i, elementKey, err)
			}
			currentBucketSize = currentBucketSize + 1
			continue
		}

		transactions[i] = mutates

		if firstBucketStart == bufferSize {
			firstBucketStart = i
		}

		if currentBucketSize > largestBucketSize {
			largestBucketSize = currentBucketSize
			largestBucketStart = currentBucketStart
		}

		currentBucketSize = 0
		currentBucketStart = i
	}

	var buffPos uint64

	if firstBucketStart == bufferSize {
		jww.WARN.Printf("All buffer buckets are full, buffer starting " +
			"position of 0 will overwrite")
	} else {
		if currentBucketSize+currentBucketStart == bufferSize-1 {
			currentBucketSize += firstBucketStart
		}
		if currentBucketSize > largestBucketSize {
			largestBucketSize = currentBucketSize
			largestBucketStart = currentBucketStart
		}
		buffPos = uint64(largestBucketStart)
	}

	mb := &mutateBuffer{
		index: &buffPos,
		kv:    kv,
	}

	return mb, transactions
}

func (mb *mutateBuffer) DoTransactionAndWriteToBuffer(op ekv.TransactionOperation,
	keys []string, mutates map[string]Mutate) (int, error) {

	mutateData, panicErr := json.Marshal(&mutates)
	if panicErr != nil {
		jww.FATAL.Panicf("Failed to json mutates: %+v", panicErr)
	}

	index := mb.getIndex()
	mutateElementKey := createElementKey(index)
	keys = append(keys, mutateElementKey)

	opWrap := func(files map[string]ekv.Operable, ext ekv.Extender) error {

		err := op(files, ext)
		if err != nil {
			return err
		}
		mutateFile := files[mutateElementKey]
		mutateFile.Set(mutateData)
		return nil
	}

	return index, mb.kv.Transaction(opWrap, keys...)
}

func (mb *mutateBuffer) DeleteBufferElement(index int) {
	mutateElementKey := createElementKey(index)
	if err := mb.kv.Delete(mutateElementKey); err != nil {
		jww.WARN.Printf("Failed to delete mutate buffer element "+
			"%d(%s): %+v", index, mutateElementKey, err)
	}
}

func (mb *mutateBuffer) getIndex() int {
	newVal := atomic.AddUint64(mb.index, 1)
	oldVal := (newVal - 1) % bufferSize
	return int(oldVal)
}

func createElementKey(index int) string {
	return buffElementKey + strconv.Itoa(int(index))
}

func msmm(k string, m Mutate) map[string]Mutate {
	mp := make(map[string]Mutate)
	mp[k] = m
	return mp
}
