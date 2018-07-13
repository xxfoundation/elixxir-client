package io

import (
	"testing"
	"gitlab.com/privategrity/client/parse"
	"math/rand"
	"gitlab.com/privategrity/crypto/format"
	"time"
)

func TestCollator_AddMessage(t *testing.T) {
	collator := &collator{
		pendingMessages: make(map[string]*multiPartMessage),
	}
	var bodies []string
	for length := uint64(5); length < 20 * format.DATA_LEN; length += 20 {
		newBody := make([]byte, length)
		_, err := rand.Read(newBody)
		if err != nil {
			t.Errorf("Couldn't generate enough random bytes: %v", err.Error())
		}
		bodies = append(bodies, string(newBody))
	}
	for i := range bodies {
		partitions, err := parse.Partition([]byte(bodies[i]), []byte{5})
		if err != nil {
			t.Errorf("Error partitioning messages: %v", err.Error())
		}
		var result []byte
		for j := range partitions {
			result = collator.AddMessage(partitions[j], 5, time.Minute)
		}
		if string(result) != bodies[i] {
			t.Errorf("Input didn't match output for %v. Got: %q, expected %q",
				i, result, bodies[i])
		}
	}
}

func TestCollator_AddMessage_Timeout(t *testing.T) {
	collator := &collator{
		pendingMessages: make(map[string]*multiPartMessage),
	}
	//enough for four partitions
	body := make([]byte, 3 * format.DATA_LEN)
	partitions, err := parse.Partition(body, []byte{88})
	if err != nil {
		t.Errorf("Error partitioning messages: %v", err.Error())
	}
	var result []byte
	for i := range partitions {
		result = collator.AddMessage(partitions[i], 88, 80 * time.Millisecond)
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
