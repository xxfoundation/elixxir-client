////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

// Unit test
func TestNewSender(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	params := DefaultParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	_, err := NewSender(params, rng, testNdf, manager, testStorage)
	if err != nil {
		t.Fatalf("Failed to create mock sender: %v", err)
	}
}

// Unit test
func TestSender_SendToAny(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	params := DefaultParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

	senderFace, err := NewSender(
		params, rng, testNdf, manager, testStorage)
	s := senderFace.(*sender)
	if err != nil {
		t.Fatalf("Failed to create mock sender: %v", err)
	}

	stream := rng.GetStream()
	defer stream.Close()

	// Put 3 gateways into the pool
	for i := 0; i < cap(s.writePool.hostList); i++ {
		gw := testNdf.Gateways[i]
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		gwHost, err := manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not add mock host to manager: %+v", err)
		}

		s.writePool.addOrReplace(stream, gwHost)
	}

	s.writePool.isConnected = func(host *connect.Host) bool { return true }

	//update the read pool
	s.readPool.Store(s.writePool)

	// Test sendToAny with test interfaces
	result, err := s.SendToAny(SendToAnyHappyPath, nil)
	if err != nil {
		t.Errorf("Should not error in SendToAny happy path: %+v", err)
	}

	if !reflect.DeepEqual(result, happyPathReturn) {
		t.Errorf("Expected result not returnev via SendToAny interface."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", happyPathReturn, result)
	}

	_, err = s.SendToAny(SendToAnyKnownError, nil)
	if err == nil {
		t.Fatalf("Expected error path did not receive error")
	}

	_, err = s.SendToAny(SendToAnyUnknownError, nil)
	if err == nil {
		t.Fatalf("Expected error path did not receive error")
	}

}

// Unit test
func TestSender_SendToPreferred(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	params := DefaultParams()
	params.PoolSize = uint32(len(testNdf.Gateways)) - 5

	// Do not test proxy attempts code in this test
	// (self contain to code specific in sendPreferred)
	params.ProxyAttempts = 0

	sFace, err := NewSender(params, rng, testNdf, manager, testStorage)
	if err != nil {
		t.Fatalf("Failed to create mock sender: %v", err)
	}
	s := sFace.(*sender)

	stream := rng.GetStream()
	defer stream.Close()

	var preferredHost *connect.Host

	// Put 3 gateways into the pool
	for i := 0; i < cap(s.writePool.hostList); i++ {
		gw := testNdf.Gateways[i]
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		gwHost, err := manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not add mock host to manager: %+v", err)
		}

		s.writePool.addOrReplace(stream, gwHost)

		if i == 1 {
			preferredHost = gwHost
		}
	}

	s.writePool.isConnected = func(host *connect.Host) bool { return true }

	//update the read pool
	s.readPool.Store(s.writePool)

	// Happy path
	result, err := s.SendToPreferred([]*id.ID{preferredHost.GetId()},
		SendToPreferredHappyPath, nil, 250*time.Millisecond)
	if err != nil {
		t.Errorf("Should not error in SendToPreferred happy path: %v", err)
	}

	if !reflect.DeepEqual(result, happyPathReturn) {
		t.Errorf("Expected result not returnev via SendToPreferred interface."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", happyPathReturn, result)
	}

	// Call a send which returns an error which triggers replacement
	_, err = s.SendToPreferred([]*id.ID{preferredHost.GetId()},
		SendToPreferredKnownError, nil, 250*time.Millisecond)
	if err == nil {
		t.Fatalf("Expected error path did not receive error")
	}

	// Check the host removal signal has been sent
	select {
	case removeSignal := <-s.removeRequest:
		if !removeSignal.Cmp(preferredHost.GetId()) {
			t.Errorf("Expected host %s to be removed due "+
				"to error, instead %s removed", preferredHost, removeSignal)
		}
	default:
		t.Errorf("Expected host %s error to trigger a removal signal",
			preferredHost)
	}

	// get a new host to test on
	preferredIndex := 4
	preferredHost = s.writePool.hostList[preferredIndex]

	// Unknown error return will not trigger replacement
	_, err = s.SendToPreferred([]*id.ID{preferredHost.GetId()},
		SendToPreferredUnknownError, nil, 250*time.Millisecond)
	if err == nil {
		t.Fatalf("Expected error path did not receive error")
	}

	// Check the host has not been replaced
	if _, ok := s.writePool.hostMap[*preferredHost.GetId()]; !ok {
		t.Errorf("Host %s should not have been removed due on an unknown error",
			preferredHost)
	}

	// Ensure we are disconnected from the old host
	if isConnected, _ := preferredHost.Connected(); isConnected {
		t.Errorf("Should not disconnect from  %s", preferredHost)
	}
}
