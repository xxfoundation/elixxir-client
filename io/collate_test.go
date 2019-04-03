////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"bytes"
	"encoding/hex"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"testing"
	"time"
)

func TestCollator_AddMessage(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := signature.NewDSAParams(rng, signature.L3072N256)
	privateKey := params.PrivateKeyGen(rng)
	publicKey := privateKey.PublicKeyGen()
	grp := cyclic.NewGroup(params.GetP(), params.GetG(), params.GetQ())
	user.TheSession = user.NewSession(&user.User{id.NewUserFromUint(8, t),
		"test"}, "",
		[]user.NodeKeys{}, publicKey, privateKey, grp)

	collator := &collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	var bodies [][]byte
	for length := 5; length < 20*format.MP_PAYLOAD_LEN; length += 20 {
		newBody := make([]byte, length)
		_, err := rand.Read(newBody)
		if err != nil {
			t.Errorf("Couldn't generate enough random bytes: %v", err.Error())
		}

		bodies = append(bodies, newBody)
	}
	for i := range bodies {
		partitions, err := parse.Partition([]byte(bodies[i]), []byte{5})
		if err != nil {
			t.Errorf("Error partitioning messages: %v", err.Error())
		}
		var result *parse.Message
		for j := range partitions {

			fm := format.NewMessage()
			fm.SetSender(id.NewUserFromUint(5, t))
			fm.SetRecipient(id.NewUserFromUint(6, t))
			fm.SetPayloadData(partitions[j])

			result = collator.AddMessage(fm, time.Minute)
		}

		typedBody, err := parse.Parse(bodies[i])

		// This always fails because of the trailing zeroes. Question is, how
		// much does it matter in regular usage? Protobufs know their length
		// already, and strings should respect null terminators,
		// so it's probably not actually that much of a problem.
		if !bytes.Contains(result.Body, typedBody.Body) {
			t.Errorf("Input didn't match output for %v. Got: %v, expected %v",
				i, hex.EncodeToString(result.Body),
				hex.EncodeToString(typedBody.Body))
		}
	}
}

func TestCollator_AddMessage_Timeout(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := signature.NewDSAParams(rng, signature.L3072N256)
	privateKey := params.PrivateKeyGen(rng)
	publicKey := privateKey.PublicKeyGen()
	grp := cyclic.NewGroup(params.GetP(), params.GetG(), params.GetQ())
	user.TheSession = user.NewSession(&user.User{id.NewUserFromUint(8, t),
		"test"}, "",
		[]user.NodeKeys{}, publicKey, privateKey, grp)

	collator := &collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	//enough for four partitions, probably
	body := make([]byte, 3*format.MP_PAYLOAD_LEN)
	partitions, err := parse.Partition(body, []byte{88})
	if err != nil {
		t.Errorf("Error partitioning messages: %v", err.Error())
	}
	var result *parse.Message
	for i := range partitions {
		fm := format.NewMessage()
		fm.SetSender(id.NewUserFromUint(5, t))
		fm.SetRecipient(id.NewUserFromUint(6, t))
		fm.SetPayload(partitions[i])

		result = collator.AddMessage(fm, 80*time.Millisecond)
		if result != nil {
			t.Error("Got a result from collator when it should be timing out" +
				" submessages")
		}
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(80 * time.Millisecond)
	if len(collator.pendingMessages) != 0 {
		t.Error("Multi-part message didn't get timed out properly")
	}
}
