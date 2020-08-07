////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
	"time"
)

func initTest(t *testing.T) *Session {
	err := os.RemoveAll(".session_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}
	s, err := Init(".session_testdir", "test")
	if err != nil {
		t.Log(s)
		t.Errorf("failed to init: %+v", err)
	}
	return s
}

// Smoke test for session object init/set/get methods
func TestSession_Smoke(t *testing.T) {
	s := initTest(t)

	err = s.Set("testkey", &VersionedObject{
		Version:   0,
		Timestamp: time.Now(),
		Data:      []byte("test"),
	})
	if err != nil {
		t.Errorf("Failed to set: %+v", err)
	}
	o, err := s.Get("testkey")
	if err != nil {
		t.Errorf("Failed to get key")
	}
	if o == nil {
		t.Errorf("Got nil return from get")
	}

	if bytes.Compare(o.Data, []byte("test")) != 0 {
		t.Errorf("Failed to get data")
	}
}

// Happy path for getting/setting LastMessageID
func TestSession_GetSetLastMessageId(t *testing.T) {
	testId := "testLastMessageId"

	s := initTest(t)

	err := s.SetLastMessageId(testId)
	if err != nil {
		t.Errorf("Failed to set LastMessageId: %+v", err)
	}
	o, err := s.GetLastMessageId()
	if err != nil {
		t.Errorf("Failed to get LastMessageId")
	}

	if testId != o {
		t.Errorf("Failed to get LastMessageID, Got %s Expected %s", o, testId)
	}
}

// Happy path for getting/setting node keys
func TestSession_GetPushNodeKeys(t *testing.T) {
	s := initTest(t)

	testId := id.NewIdFromString("test", id.Node, t)
	testId2 := id.NewIdFromString("test2", id.Node, t)
	testInt := cyclic.NewGroup(large.NewIntFromUInt(6), large.NewIntFromUInt(6)).NewInt(1)
	testNodeKey := user.NodeKeys{
		TransmissionKey: testInt,
		ReceptionKey:    testInt,
	}

	err := s.PushNodeKey(testId, testNodeKey)
	if err != nil {
		t.Errorf("Unable to push node key: %+v", err)
	}
	err = s.PushNodeKey(testId2, testNodeKey)
	if err != nil {
		t.Errorf("Unable to push node key: %+v", err)
	}

	circ := connect.NewCircuit([]*id.ID{testId, testId2})
	results, err := s.GetNodeKeysFromCircuit(circ)

	if len(results) != 2 {
		t.Errorf("Returned unexpected number of node keys: %d", len(results))
		return
	}
	if results[0].TransmissionKey.Cmp(testInt) != 0 {
		t.Errorf("Returned invalid transmission key: %s, Expected: %s", results[0].TransmissionKey.Text(10),
			testInt.Text(10))
	}
	if results[0].ReceptionKey.Cmp(testInt) != 0 {
		t.Errorf("Returned invalid reception key: %s, Expected: %s", results[0].TransmissionKey.Text(10),
			testInt.Text(10))
	}
}
