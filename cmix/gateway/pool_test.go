////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// Unit test, happy paths of GetAny.
func TestHostPool_GetAny(t *testing.T) {
	manager := newMockManager()
	rng := rand.New(rand.NewSource(42))
	testNdf := getTestNdf(t)
	params := DefaultParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	// Call the constructor
	testPool := newPool(5)

	// Pull all gateways from NDF into host manager
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		var h *connect.Host
		h, err = manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not Add mock host to manager: %+v", err)
		}

		//Add to the host pool
		testPool.addOrReplace(rng, h)

	}

	testPool.isConnected = func(host *connect.Host) bool { return true }

	requested := 3
	anyList := testPool.GetAny(uint32(requested), nil, rng)
	if len(anyList) != requested {
		t.Errorf("GetAnyList did not get requested length."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", requested, len(anyList))
	}

	for _, h := range anyList {
		_, ok := manager.GetHost(h.GetId())
		if !ok {
			t.Errorf("Host %s in retrieved list not in manager", h)
		}
	}

	// Request more than are in host list
	largeRequest := uint32(requested * 1000)
	largeRetrieved := testPool.GetAny(largeRequest, nil, rng)
	if len(largeRetrieved) != len(testPool.hostList) {
		t.Errorf("Large request should result in a list of all in host list")
	}

	// request the whole host pool with a member exluced
	excluded := []*id.ID{testPool.hostList[2].GetId()}
	requestedExcluded := uint32(len(testPool.hostList))
	excludedRetrieved := testPool.GetAny(requestedExcluded, excluded, rng)

	if len(excludedRetrieved) != int(requestedExcluded-1) {
		t.Errorf("One member should not have been returned due to being excluded")
	}

	for i := 0; i < len(excludedRetrieved); i++ {
		if excludedRetrieved[i].GetId().Cmp(excluded[0]) {
			t.Errorf("index %d of the returned list includes the excluded id %s", i, excluded[0])
		}
	}
}

// Unit test, happy paths of GetAny.
func TestHostPool_GetSpecific(t *testing.T) {
	manager := newMockManager()
	rng := rand.New(rand.NewSource(42))
	testNdf := getTestNdf(t)
	params := DefaultParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	// Call the constructor
	poolLen := 5
	testPool := newPool(poolLen)

	testPool.isConnected = func(host *connect.Host) bool { return true }

	// Pull all gateways from NDF into host manager
	for i, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		var h *connect.Host
		h, err = manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not Add mock host to manager: %+v", err)
		}

		//Add to the host pool
		if i < poolLen {
			testPool.addOrReplace(rng, h)
		}
	}

	testPool.isConnected = func(host *connect.Host) bool { return true }

	//test get specific returns something in the host pool]
	toGet := testPool.hostList[0].GetId()
	h, exists := testPool.GetSpecific(toGet)
	if !exists {
		t.Errorf("Failed to get member of host pool that should "+
			"be there, got: %s", h)
	}
	if h == nil || !h.GetId().Cmp(toGet) {
		t.Errorf("Wrong or invalid host returned")
	}

	//test get specific returns nothing when the item is not in the host pool
	toGet, _ = testNdf.Gateways[poolLen+1].GetGatewayId()
	h, exists = testPool.GetSpecific(toGet)
	if exists || h != nil {
		t.Errorf("Got a member of host pool that should not be there")
	}

}

// Full test
func TestHostPool_GetPreferred(t *testing.T) {
	manager := newMockManager()
	rng := rand.New(rand.NewSource(42))
	testNdf := getTestNdf(t)
	params := DefaultParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

	poolLen := 12
	testPool := newPool(poolLen)

	testPool.isConnected = func(host *connect.Host) bool { return true }

	// Pull all gateways from NDF into host manager
	hostMap := make(map[id.ID]bool, 0)
	targets := make([]*id.ID, 0)
	for i, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		h, err := manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not Add mock host to manager: %+v", err)
		}

		hostMap[*gwId] = true
		targets = append(targets, gwId)

		//Add to the host pool
		if i < poolLen {
			testPool.addOrReplace(rng, h)
		}

	}

	retrievedList := testPool.GetPreferred(targets, rng)
	if len(retrievedList) != len(targets) {
		t.Errorf("Requested list did not output requested length."+
			"\n\tExpected: %d"+
			"\n\tReceived: %v", len(targets), len(retrievedList))
	}

	// In case where all requested gateways are present
	// ensure requested hosts were returned
	for _, h := range retrievedList {
		if !hostMap[*h.GetId()] {
			t.Errorf("A target gateways which should have been returned was not."+
				"\n\tExpected: %v", h.GetId())
		}
	}

	// Replace a request with a gateway not in pool
	targets[3] = id.NewIdFromUInt(74, id.Gateway, t)
	retrievedList = testPool.GetPreferred(targets, rng)
	if len(retrievedList) != len(targets) {
		t.Errorf("Requested list did not output requested length."+
			"\n\tExpected: %d"+
			"\n\tReceived: %v", len(targets), len(retrievedList))
	}

	// In case where a requested gateway is not present
	for _, h := range retrievedList {
		if h.GetId().Cmp(targets[3]) {
			t.Errorf("Should not have returned ID not in pool")
		}
	}

}
