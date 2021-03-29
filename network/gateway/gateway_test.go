///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Full happy path test
func TestReplaceHost(t *testing.T)  {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:      manager,
		hostList:     make([]*connect.Host, newIndex+1),
		hostMap:      make(map[id.ID]uint32),
		ndf:          testNdf,
	}

	/* "Replace" a host with no entry */

	// Pull a gateway ID from the ndf
	gwIdOne, err := id.Unmarshal(testNdf.Gateways[0].ID)
	if err != nil {
		t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
	}

	// Add mock gateway to manager
	_, err = manager.AddHost(gwIdOne, "", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Could not add mock host to manager: %v", err)
	}

	// "Replace" (insert) the host
	err = hostPool.replaceHost(gwIdOne, newIndex)
	if err != nil {
		t.Errorf("Could not replace host: %v", err)
	}

	// Check the state of the map has been correctly updated
	retrievedIndex, ok := hostPool.hostMap[*gwIdOne]
	if !ok {
		t.Errorf("Expected insertion of gateway ID into map")
	}
	if retrievedIndex != newIndex {
		t.Errorf("Index pulled from map not expected value." +
			"\n\tExpected: %d" +
			"\n\tReceived: %d", newIndex, retrievedIndex)
	}

	// Check that the state of the list list been correctly updated
	retrievedHost := hostPool.hostList[newIndex]
	if !gwIdOne.Cmp(retrievedHost.GetId()) {
		t.Errorf("Id pulled from list is not expected." +
			"\n\tExpected: %s" +
			"\n\tReceived: %s", gwIdOne, retrievedHost.GetId())
	}

	/* Replace the initial host with a new host */

	// Pull a different gateway ID from the ndf
	gwIdTwo, err := id.Unmarshal(testNdf.Gateways[1].ID)
	if err != nil {
		t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
	}

	// Add second mock gateway to manager
	_, err = manager.AddHost(gwIdTwo, "", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Could not add mock host to manager: %v", err)
	}


	// Replace the old host
	err = hostPool.replaceHost(gwIdTwo, newIndex)
	if err != nil {
		t.Errorf("Could not replace host: %v", err)
	}

	// Check that the state of the list been correctly updated for new host
	retrievedHost = hostPool.hostList[newIndex]
	if !gwIdTwo.Cmp(retrievedHost.GetId()) {
		t.Errorf("Id pulled from list is not expected." +
			"\n\tExpected: %s" +
			"\n\tReceived: %s", gwIdTwo, retrievedHost.GetId())
	}

	// Check the state of the map has been correctly removed for the old gateway
	retrievedOldIndex, ok := hostPool.hostMap[*gwIdOne]
	if ok {
		t.Errorf("Exoected old gateway to be cleared from map")
	}
	if retrievedOldIndex != 0 {
		t.Errorf("Index pulled from map with old gateway as the key " +
			"was not cleared." +
			"\n\tExpected: %d" +
			"\n\tReceived: %d", 0, retrievedOldIndex)
	}


	// Check the state of the map has been correctly updated for the old gateway
	retrievedIndex, ok = hostPool.hostMap[*gwIdTwo]
	if !ok {
		t.Errorf("Expected insertion of gateway ID into map")
	}
	if retrievedIndex != newIndex {
		t.Errorf("Index pulled from map using new gateway as the key " +
			"was not updated." +
			"\n\tExpected: %d" +
			"\n\tReceived: %d", newIndex, retrievedIndex)
	}
}

// Error path, could not get host
func TestReplaceHost_Error(t *testing.T)  {
	manager := newHappyManager()

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:        manager,
		hostList:     make([]*connect.Host, 1),
		hostMap:      make(map[id.ID]uint32),
	}

	// Construct an unknown gateway ID to the manager
	gatewayId := id.NewIdFromString("BadGateway", id.Gateway, t)

	err := hostPool.replaceHost(gatewayId, 0)
	if err == nil {
		t.Errorf("Expected error in happy path: Should not be able to find a host")
	}

}

// Happy path
func TestPruneHostPool(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways))
	rng := csprng.NewSystemRNG()

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:      manager,
		hostList:     make([]*connect.Host, newIndex+1),
		hostMap:      make(map[id.ID]uint32),
		ndf:          testNdf,
		poolParams: params,
		rng: rng,
	}

	// Pull all gateways from ndf into host manager
	hostList := make([]*connect.Host, 0)
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
		}
		// Add mock gateway to manager
		h, err := manager.AddHost(gwId, "", nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Errorf("Could not add mock host to manager: %v", err)
			t.FailNow()
		}


		hostList = append(hostList, h)

	}
	// fixme: have a case where host's error threshold is met?

	// Call prune host pool
	err := hostPool.pruneHostPool()
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v",err)
	}

	// Check that the host map has been properly updated
	for _, h := range hostList {
		_, ok := hostPool.hostMap[*h.GetId()]
		if !ok {
			t.Errorf("Gateway %s was not placed in host map after pruning", h.GetId().String())
		}
	}

}

// Error path: not enough gateways in ndf compared to
// required pool size
func TestPruneHostPool_Error(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()

	// Trigger the case where the Ndf doesn't have enough gateways
	params.poolSize = uint32(len(testNdf.Gateways)) + 1
	rng := csprng.NewSystemRNG()

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:      manager,
		hostList:     make([]*connect.Host, newIndex+1),
		hostMap:      make(map[id.ID]uint32),
		ndf:          testNdf,
		poolParams: params,
		rng: rng,
	}

	// Call prune
	err := hostPool.pruneHostPool()
	if err == nil {
		t.Errorf("Gateways should not be available: " +
			"not enough gateways in ndf compared to param's pool size")
	}

}