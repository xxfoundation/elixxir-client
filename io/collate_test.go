package io

import (
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/format"
	"math/rand"
	"reflect"
	"testing"
	"time"
	"gitlab.com/privategrity/crypto/id"
)

func TestCollator_AddMessage(t *testing.T) {

	user.TheSession = user.NewSession(&user.User{id.NewUserIDFromUint(8, t),
	"test"}, "",
		[]user.NodeKeys{})

	collator := &collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	var bodies [][]byte
	for length := uint64(5); length < 20*format.DATA_LEN; length += 20 {
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

			fm, errFNM := format.NewMessage(id.NewUserIDFromUint(5, t),
				id.NewUserIDFromUint(6, t), string(partitions[j]))

			if errFNM != nil {
				t.Errorf("Collator.AddMessage: Failed to format valid message: %s", errFNM.Error())
			}

			result = collator.AddMessage(&fm[0], time.Minute)
		}

		typedBody, err := parse.Parse(bodies[i])

		if !reflect.DeepEqual(result.Body, typedBody.Body) {
			t.Errorf("Input didn't match output for %v. Got: %v, expected %v",
				i, result.Body, typedBody.Body)
		}
	}
}

func TestCollator_AddMessage_Timeout(t *testing.T) {

	user.TheSession = user.NewSession(&user.User{id.NewUserIDFromUint(8, t),
	"test"}, "",
		[]user.NodeKeys{})

	collator := &collator{
		pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
	}
	//enough for four partitions
	body := make([]byte, 3*format.DATA_LEN)
	partitions, err := parse.Partition(body, []byte{88})
	if err != nil {
		t.Errorf("Error partitioning messages: %v", err.Error())
	}
	var result *parse.Message
	for i := range partitions {
		fm, errFNM := format.NewMessage(id.NewUserIDFromUint(5, t),
			id.NewUserIDFromUint(6, t), string(partitions[i]))

		if errFNM != nil {
			t.Errorf("Collator.AddMessage: Failed to format valid message: %s", errFNM.Error())
		}

		result = collator.AddMessage(&fm[0], 80*time.Millisecond)
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
