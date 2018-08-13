////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"reflect"
	"testing"
)

//Shows that MessageHash ia an independent function of every field in Message
func TestMessage_Hash(t *testing.T) {
	m := Message{}
	m.Type = 0
	m.Body = []byte{0, 0}
	m.Sender = 0
	m.Receiver = 0
	m.Nonce = []byte{0, 0}

	baseHash := m.Hash()

	m.Type = 1

	typeHash := m.Hash()

	if reflect.DeepEqual(baseHash, typeHash) {
		t.Errorf("Message.Hash: Output did not change with modified type")
	}

	m.Type = 0

	m.Body = []byte{1, 1}

	bodyHash := m.Hash()

	if reflect.DeepEqual(baseHash, bodyHash) {
		t.Errorf("Message.Hash: Output did not change with modified body")
	}

	m.Body = []byte{0, 0}

	m.Sender = 1

	senderHash := m.Hash()

	if reflect.DeepEqual(baseHash, senderHash) {
		t.Errorf("Message.Hash: Output did not change with modified sender")
	}

	m.Sender = 0

	m.Receiver = 1

	receiverHash := m.Hash()

	if reflect.DeepEqual(baseHash, receiverHash) {
		t.Errorf("Message.Hash: Output did not change with modified receiver")
	}

	m.Receiver = 0

	// FIXME Add a "bake" step to the message to partition and nonceify it
	// before hashing. We need this to be able to identify messages by their
	// hash on both the message's sending and receiving clients.
	//m.Nonce = []byte{1, 1}
	//
	//nonceHash := m.Hash()
	//
	//if reflect.DeepEqual(baseHash, nonceHash) {
	//	t.Errorf("Message.Hash: Output did not change with modified nonce")
	//}
	//
	//m.Nonce = []byte{0, 0}
}
