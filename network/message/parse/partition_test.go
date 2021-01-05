package parse

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

var ipsumTestStr = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras sit amet euismod est. Donec dolor " +
	"neque, efficitur et interdum eu, lacinia quis mi. Duis bibendum elit ac lacus finibus pharetra. Suspendisse " +
	"blandit erat in odio faucibus consectetur. Suspendisse sed consequat purus. Curabitur fringilla mi sit amet odio " +
	"interdum suscipit. Etiam vitae dui posuere, congue mi a, convallis odio. In commodo risus at lorem volutpat " +
	"placerat. In cursus magna purus, suscipit dictum lorem aliquam non. Praesent efficitur."

// Test that NewPartitioner outputs a correctly formatted Partitioner
func TestNewPartitioner(t *testing.T) {
	storeSession := storage.InitTestingSession(t)
	p := NewPartitioner(4096, storeSession)

	if p.baseMessageSize != 4096 {
		t.Errorf("baseMessageSize content mismatch"+
			"\n\texpected: %v\n\treceived: %v",
			4096, p.baseMessageSize)
	}

	if p.deltaFirstPart != 20 {
		t.Errorf("deltaFirstPart content mismatch"+
			"\n\texpected: %v\n\treceived: %v",
			20, p.deltaFirstPart)
	}

	if p.firstContentsSize != 4069 {
		t.Errorf("firstContentsSize content mismatch"+
			"\n\texpected: %v\n\treceived: %v",
			4069, p.firstContentsSize)
	}

	if p.maxSize != 1042675 {
		t.Errorf("maxSize content mismatch"+
			"\n\texpected: %v\n\treceived: %v",
			1042675, p.maxSize)
	}

	if p.partContentsSize != 4089 {
		t.Errorf("partContentsSize content mismatch"+
			"\n\texpected: %v\n\treceived: %v",
			4089, p.partContentsSize)
	}

	if p.session != storeSession {
		t.Errorf("session content mismatch")
	}
}

func TestPartitioner_Partition(t *testing.T) {
	storeSession := storage.InitTestingSession(t)
	p := NewPartitioner(len(ipsumTestStr), storeSession)

	_, _, err := p.Partition(&id.DummyUser, message.Text,
		time.Now(), []byte(ipsumTestStr))
	if err != nil {
		t.Error(err)
	}
}

func TestPartitioner_HandlePartition(t *testing.T) {
	storeSession := storage.InitTestingSession(t)
	p := NewPartitioner(len(ipsumTestStr), storeSession)

	m := newMessagePart(1107, 1, []byte(ipsumTestStr))

	_, _ = p.HandlePartition(
		&id.DummyUser,
		message.None,
		m.Bytes(),
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g',
			's', 't', 'r', 'i', 'n', 'g'},
	)
}
