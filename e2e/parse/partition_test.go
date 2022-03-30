///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package parse

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
)

var ipsumTestStr = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras sit amet euismod est. Donec dolor " +
	"neque, efficitur et interdum eu, lacinia quis mi. Duis bibendum elit ac lacus finibus pharetra. Suspendisse " +
	"blandit erat in odio faucibus consectetur. Suspendisse sed consequat purus. Curabitur fringilla mi sit amet odio " +
	"interdum suscipit. Etiam vitae dui posuere, congue mi a, convallis odio. In commodo risus at lorem volutpat " +
	"placerat. In cursus magna purus, suscipit dictum lorem aliquam non. Praesent efficitur."

// Test that NewPartitioner outputs a correctly made Partitioner
func TestNewPartitioner(t *testing.T) {
	p := NewPartitioner(4096, versioned.NewKV(make(ekv.Memstore)))

	if p.baseMessageSize != 4096 {
		t.Errorf("baseMessageSize content mismatch."+
			"\nexpected: %d\nreceived: %d", 4096, p.baseMessageSize)
	}

	if p.deltaFirstPart != firstHeaderLen-headerLen {
		t.Errorf("deltaFirstPart content mismatch.\nexpected: %d\nreceived: %d",
			firstHeaderLen-headerLen, p.deltaFirstPart)
	}

	if p.firstContentsSize != 4096-firstHeaderLen {
		t.Errorf("firstContentsSize content mismatch."+
			"\nexpected: %d\nreceived: %d",
			4096-firstHeaderLen, p.firstContentsSize)
	}

	if p.maxSize != (4096-firstHeaderLen)+(MaxMessageParts-1)*(4096-headerLen) {
		t.Errorf("maxSize content mismatch.\nexpected: %d\nreceived: %d",
			(4096-firstHeaderLen)+(MaxMessageParts-1)*(4096-headerLen),
			p.maxSize)
	}

	if p.partContentsSize != 4088 {
		t.Errorf("partContentsSize content mismatch."+
			"\nexpected: %d\nreceived: %d", 4088, p.partContentsSize)
	}

}

// Test that no error is returned running Partitioner.Partition.
func TestPartitioner_Partition(t *testing.T) {
	p := NewPartitioner(len(ipsumTestStr), versioned.NewKV(make(ekv.Memstore)))

	_, _, err := p.Partition(
		&id.DummyUser, catalog.XxMessage, netTime.Now(), []byte(ipsumTestStr))
	if err != nil {
		t.Error(err)
	}
}

// Test that Partitioner.HandlePartition can handle a message part.
func TestPartitioner_HandlePartition(t *testing.T) {
	p := NewPartitioner(len(ipsumTestStr), versioned.NewKV(make(ekv.Memstore)))
	m := newMessagePart(1107, 1, []byte(ipsumTestStr))

	_, _ = p.HandlePartition(
		&id.DummyUser,
		m.bytes(),
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g', 's', 't', 'r', 'i', 'n', 'g'},
	)
}

// Test that HandlePartition can handle a first message part
func TestPartitioner_HandleFirstPartition(t *testing.T) {
	p := NewPartitioner(len(ipsumTestStr), versioned.NewKV(make(ekv.Memstore)))
	m := newFirstMessagePart(
		catalog.XxMessage, 1107, 1, netTime.Now(), []byte(ipsumTestStr))

	_, _ = p.HandlePartition(
		&id.DummyUser,
		m.bytes(),
		[]byte{'t', 'e', 's', 't', 'i', 'n', 'g', 's', 't', 'r', 'i', 'n', 'g'},
	)
}
