////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"os"
	"testing"
	"time"
)

// Smoke test for session object init/set/get methods
func TestSession_Smoke(t *testing.T) {
	err := os.RemoveAll(".session_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}
	s, err := Init(".session_testdir", "test")
	if err != nil {
		t.Log(s)
		t.Errorf("failed to init: %+v", err)
	}

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

	err := os.RemoveAll(".session_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}
	s, err := Init(".session_testdir", "test")
	if err != nil {
		t.Log(s)
		t.Errorf("failed to init: %+v", err)
	}

	err = s.SetLastMessageId(testId)
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
