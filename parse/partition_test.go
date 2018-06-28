package parse

import (
	"testing"
	"math/rand"
	"bytes"
	"gitlab.com/privategrity/crypto/format"
)

func randomString(seed int64, length int) []byte {
	buffer := make([]byte, length)
	rand.Seed(seed)
	rand.Read(buffer)
	return buffer
}

func TestPartitionEmptyMessage(t *testing.T) {
	id := []byte{0x05}
	actual, err := Partition(randomString(0, 0), id)
	if err != nil {
		t.Error(err.Error())
	}
	expected := [][]byte{{0x05, 0x0, 0x0},}
	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition empty message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

func TestPartitionShort(t *testing.T) {
	id := []byte{0x03}
	randomBytes := randomString(0, 50)
	actual, err := Partition(randomBytes, id)
	if err != nil {
		t.Error(err.Error())
	}
	expected := [][]byte{{0x03, 0x0, 0x0}}
	expected[0] = append(expected[0], randomBytes...)
	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition short message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

func TestPartitionLong(t *testing.T) {
	id := []byte{0xa2, 0x54}
	randomBytes := randomString(0, 300)
	actual, err := Partition(randomBytes, id)

	if err != nil {
		t.Error(err.Error())
	}

	expected := make([][]byte, 2)
	// id
	expected[0] = append(expected[0], id...)
	// index
	expected[0] = append(expected[0], 0, 1)
	// part of random string
	expected[0] = append(expected[0], randomBytes[:format.DATA_LEN-4]...)

	// id
	expected[1] = append(expected[1], id...)
	// index
	expected[1] = append(expected[1], 1, 1)
	// other part of random string
	expected[1] = append(expected[1], randomBytes[format.DATA_LEN-4:]...)

	for i := range actual {
		if !bytes.Equal(actual[i], expected[i]) {
			t.Errorf("Partition long message: expected partition %v differed"+
				" from actual partition %v", expected[i], actual[i])
		}
	}
}

func TestPartitionLongest(t *testing.T) {
	// I'm assuming that 5 bytes will be the longest possible ID because that
	// is the max length of a uvarint with 32 bits
	id := []byte{0x1f, 0x2f, 0x3f, 0x4f, 0x5f}
	actual, err := Partition(randomString(0, 57855), id)

	if err != nil {
		t.Error(err.Error())
	}

	expectedNumberOfPartitions := 256

	if len(actual) != expectedNumberOfPartitions {
		t.Errorf("Expected a 57855-byte message to split into %v partitions",
			expectedNumberOfPartitions)
	}

	// check the index and max index of the last partition
	expectedIdx := byte(255)
	idxLocation := len(id)
	maxIdxLocation := len(id) + 1
	actualIdx := actual[len(actual)-1][idxLocation]
	actualMaxIdx := actual[len(actual)-1][maxIdxLocation]
	if actualIdx != expectedIdx {
		t.Errorf("Expected index of %v on the last partition, got %v",
			expectedIdx, actualIdx)
	}
	if actualMaxIdx != expectedIdx {
		t.Errorf("Expected max index of %v on the last partition, got %v",
			expectedIdx, actualMaxIdx)
	}
}

// Tests production of the error
func TestPartitionTooLong(t *testing.T) {
	id := []byte{0x1f, 0x2f, 0x3f, 0x4f, 0x5f}
	_, err := Partition(randomString(0, 57856), id)

	if err == nil {
		t.Error("Partition() processed a message that was too long to be" +
			" partitioned")
	}
}

// Tests Assemble with a synthetic test case, without invoking Partition.
func TestOnlyAssemble(t *testing.T) {
	id := []byte{0xf0, 0xe0, 0xd0, 0xc0, 0xb0, 0xa0, 0x04}
	// Assemble ignores these. Messages should be ordered elsewhere
	indexAndMaxIndex := []byte{0x4, 0x8}

	messageChunks := []string{"Han Singular, ", "my child, ",
		"awaken and embrace ", "the glory that is", " your birthright."}

	completeMessage := ""
	for i := range messageChunks {
		completeMessage += messageChunks[i]
	}

	partitions := make([][]byte, len(messageChunks))
	for i := range partitions {
		partitions[i] = append(partitions[i], id...)
		partitions[i] = append(partitions[i], indexAndMaxIndex...)
		partitions[i] = append(partitions[i], messageChunks[i]...)
	}

	if completeMessage != string(Assemble(partitions)) {
		t.Errorf("TestOnlyAssemble: got \"%v\"; expected \"%v\".",
			string(Assemble(partitions)), completeMessage)
	}
}

func TestAssembleEmpty(t *testing.T) {

}

func TestAssembleShort(t *testing.T) {

}

func TestAssembleLong(t *testing.T) {

}

func TestAssembleLongest(t *testing.T) {

}
