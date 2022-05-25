////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package single

// // Test failure of proto unmarshal
// func TestSingleReceiver_Callback_FailUnmarshal(t *testing.T) {
// 	ep := restlike.NewEndpoints()
// 	r := receiver{endpoints: ep}

// 	testReq := single.BuildTestRequest(make([]byte, 0), t)
// 	r.Callback(testReq, receptionID.EphemeralIdentity{}, nil)
// }

// Test happy path
//func TestSingleReceiver_Callback(t *testing.T) {
//	ep := &Endpoints{endpoints: make(map[URI]map[Method]Callback)}
//	resultChan := make(chan interface{}, 1)
//	cb := func(*Message) *Message {
//		resultChan <- ""
//		return nil
//	}
//	testPath := URI("test/path")
//	testMethod := Get
//	testMessage := &Message{
//		Content: []byte("test"),
//		Headers: nil,
//		Method:  uint32(testMethod),
//		Uri:     string(testPath),
//		Error:   "",
//	}
//
//	err := ep.Add(testPath, testMethod, cb)
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	receiver := receiver{endpoints: ep}
//
//	testPayload, err := proto.Marshal(testMessage)
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	testReq := single.BuildTestRequest(testPayload, t)
//	receiver.Callback(testReq, receptionID.EphemeralIdentity{}, nil)
//
//	select {
//	case _ = <-resultChan:
//	case <-time.After(3 * time.Second):
//		t.Errorf("Test SingleReceiver timed out!")
//	}
//}
