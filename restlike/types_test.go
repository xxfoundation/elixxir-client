////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package restlike

import "testing"

// Full test for all add/get/remove cases
func TestEndpoints(t *testing.T) {
	ep := &Endpoints{endpoints: make(map[URI]map[Method]Callback)}
	cb := func(*Message) *Message {
		return nil
	}

	testPath := URI("test/path")
	testMethod := Get
	err := ep.Add(testPath, testMethod, cb)
	if _, ok := ep.endpoints[testPath][testMethod]; err != nil || !ok {
		t.Errorf("Failed to add endpoint: %+v", err)
	}
	err = ep.Add(testPath, testMethod, cb)
	if _, ok := ep.endpoints[testPath][testMethod]; err == nil || !ok {
		t.Errorf("Expected failure to add endpoint")
	}

	resultCb, err := ep.Get(testPath, testMethod)
	if resultCb == nil || err != nil {
		t.Errorf("Expected to get endpoint: %+v", err)
	}

	err = ep.Remove(testPath, testMethod)
	if _, ok := ep.endpoints[testPath][testMethod]; err != nil || ok {
		t.Errorf("Failed to remove endpoint: %+v", err)
	}
	err = ep.Remove(testPath, testMethod)
	if _, ok := ep.endpoints[testPath][testMethod]; err == nil || ok {
		t.Errorf("Expected failure to remove endpoint")
	}

	resultCb, err = ep.Get(testPath, testMethod)
	if resultCb != nil || err == nil {
		t.Errorf("Expected failure to get endpoint: %+v", err)
	}
}
