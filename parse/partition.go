package parse

import (
	"encoding/binary"
	"gitlab.com/privategrity/crypto/format"
	"math"
	"sync"
	"errors"
)

// TODO is there a better way to generate unique message IDs locally?
// also, dummy message sender needs to have some way to get around this
type idCounter struct {
	// 32 bits to put a smaller upper bound on the varint size on the wire
	currentID uint32
	mux       sync.Mutex
}

var currentCounter idCounter

func (i *idCounter) nextID() []byte {
	// this will use up to 5 bytes for the message ID
	result := make([]byte, binary.MaxVarintLen32)
	i.mux.Lock()
	n := binary.PutUvarint(result, uint64(i.currentID))
	i.currentID++
	i.mux.Unlock()
	return result[:n]
}

func (i *idCounter) reset() {
	i.mux.Lock()
	i.currentID = 0
	i.mux.Unlock()
}

const MessageTooLongError = "Partition(): Message is too long to partition"

func Partition(body []byte, id []byte) ([][]byte, error) {
	// index and quantity of the partitioned message are a fixed length of 8
	// bits because sending more than that through the system is really slow and
	// making them variable length makes the required length of the body part
	// of the partitions different per partition depending on what the length
	// of the index is for the input message
	// the bottom line is that there's a dependency cycle to calculate the right
	// number of partitions if you do them that way and i'm having none of that

	// a zero here means that the message has one partition
	maxIndex := uint64(len(body)) / (format.DATA_LEN - uint64(len(id)))
	if maxIndex > math.MaxUint8 {
		return nil, errors.New(MessageTooLongError)
	}

	partitions := make([][]byte, maxIndex+1)
	var lastPartitionLength int
	partitionReadIdx := 0
	for i := range partitions {
		partitions[i], lastPartitionLength = makePartition(format.DATA_LEN,
			body[partitionReadIdx:], id, byte(i), byte(maxIndex))
		partitionReadIdx += lastPartitionLength
	}

	return partitions, nil
}

// can you believe that golang doesn't provide a min function in the std lib?
// neither can i
func min(a uint64, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// makePartition makes a new partition of a multi-part message and prepends the
// id, index, and length that are needed to rebuild it on the receiving client.
// It returns the new partition and the length of the body that it consumed
// when making the new partition.
func makePartition(maxLength uint64, body []byte, id []byte, i byte,
	maxIndex byte) ([]byte, int) {

	partition := make([]byte, 0, maxLength)

	// Append the front matter
	partition = append(partition, id...)
	partition = append(partition, i, maxIndex)
	lengthBeforeBodyAppend := len(partition)

	// Find the biggest part of the body that can fit into the message length
	bodyWriteLength := min(maxLength-uint64(len(partition)), uint64(len(body)))

	// Append body
	partition = append(partition, body[:bodyWriteLength]...)

	// Return new partition and number of bytes from the body that are in it
	return partition, len(partition) - lengthBeforeBodyAppend
}

// TODO use map to hold pending messages by ID and sender

// Assemble ignores message IDs and assumes that all messages have the same ID
// It also assumes that messages are already correctly ordered by their index
func Assemble(partitions [][]byte) []byte {
	// this will allocate a bit more capacity than needed but not so much that
	// it breaks the bank
	result := make([]byte, 0, int(format.DATA_LEN)*len(partitions))

	for i := range partitions {
		result = append(result, stripPartition(partitions[i])...)
	}
	return result
}

// Strips the metadata (index, length, ID) from the start of a partition
func stripPartition(partition []byte) []byte {
	// ID is first, and it's variable length
	msbMask := byte(0x80)
	indexInformationStart := 0
	for i := 0; i < len(partition); i++ {
		if msbMask&partition[i] == 0 {
			// this is the last byte in the ID. stop the loop
			indexInformationStart = i + 1
			break
		}
	}
	// 1 byte for index, 1 byte for length.
	// Change this if you change the data type for these fields
	indexInformationLength := 2
	return partition[indexInformationStart+indexInformationLength:]
}
