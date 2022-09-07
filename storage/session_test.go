////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

/*
func initTest(t *testing.T) *Session {
	err := os.RemoveAll(".session_testdir")
	if err != nil {
		t.Errorf(err.Error())
	}
	s, err := New(".session_testdir", "test")
	if err != nil {
		t.Log(s)
		t.Errorf("failed to init: %+v", err)
	}
	return s
}

// Smoke test for session object init/set/get methods
func TestSession_Smoke(t *testing.T) {
	s := initTest(t)

	err := s.Set("testkey", &versioned.Object{
		Version:   0,
		Timestamp: netTime.Now(),
		Data:      []byte("test"),
	})
	if err != nil {
		t.Errorf("Failed to set: %+v", err)
	}
	o, err := s.get("testkey")
	if err != nil {
		t.Errorf("Failed to get key")
	}
	if o == nil {
		t.Errorf("Got nil return from get")
	}

	if bytes.Compare(o.Data, []byte("test")) != 0 {
		t.Errorf("Failed to get data")
	}
}*/
