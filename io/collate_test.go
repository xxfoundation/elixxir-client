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
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"math/rand"
	"testing"
	"time"
)

func TestCollator_AddMessage(t *testing.T) {

	uid := id.NewIdFromUInt(69, id.User, t)

	collator := &Collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	var bodies [][]byte
	for length := 5; length < 20*format.TotalLen; length += 20 {
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
			fm.SetRecipient(id.NewIdFromUInt(6, id.User, t))
			fm.Contents.SetRightAligned(partitions[j])

			result, err = collator.AddMessage(fm, uid, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
		}

		typedBody, err := parse.Parse(bodies[i])

		// This always fails because of the trailing zeroes. Question is, how
		// much does it matter in regular usage? Protobufs know their length
		// already, and strings should respect null terminators,
		// so it's probably not actually that much of a problem.
		if !bytes.Contains(result.Body, typedBody.Body) {
			t.Errorf("Input didn't match output for %v. \n  Got: %v\n  Expected %v",
				i, hex.EncodeToString(result.Body),
				hex.EncodeToString(typedBody.Body))
		}
	}
}

func TestCollator_AddMessage_Timeout(t *testing.T) {

	uid := id.NewIdFromUInt(69, id.User, t)

	collator := &Collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	//enough for four partitions, probably
	body := make([]byte, 3*format.ContentsLen)
	partitions, err := parse.Partition(body, []byte{88})
	if err != nil {
		t.Errorf("Error partitioning messages: %v", err.Error())
	}
	var result *parse.Message
	for i := range partitions {
		fm := format.NewMessage()
		now := time.Now()
		nowBytes, _ := now.MarshalBinary()
		nowBytes = append(nowBytes, make([]byte, format.TimestampLen-len(nowBytes))...)
		fm.SetTimestamp(nowBytes)
		fm.SetRecipient(id.NewIdFromUInt(6, id.User, t))
		fm.Contents.SetRightAligned(partitions[i])

		result, err = collator.AddMessage(fm, uid, 80*time.Millisecond)
		if err != nil {
			t.Fatal(err)
		}
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
