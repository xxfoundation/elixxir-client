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

	ts, err := time.Now().MarshalText()
	if err != nil {
		t.Errorf("Failed to martial time for object")
	}
	err = s.Set("testkey", &VersionedObject{
		Version:   0,
		Timestamp: ts,
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
