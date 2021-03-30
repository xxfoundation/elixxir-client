///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package gateway

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"reflect"
	"testing"
)

// Unit test
func TestNewHostPool(t *testing.T) {
	manager := newHappyManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways))

	// Pull all gateways from ndf into host manager
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
		}
		// Add mock gateway to manager
		_, err = manager.AddHost(gwId, "", nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Errorf("Could not add mock host to manager: %v", err)
			t.FailNow()
		}

	}

	// Call the constructor
	_, err := NewHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}
}

// Unit test
func TestHostPool_UpdateNdf(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:  manager,
		hostList: make([]*connect.Host, newIndex+1),
		hostMap:  make(map[id.ID]uint32),
		ndf:      testNdf,
	}

	// Construct a new Ndf different from original one above
	newNdf := getTestNdf(t)
	newGateway := ndf.Gateway{
		ID:             id.NewIdFromUInt(27, id.Gateway, t).Bytes(),
	}
	newNdf.Gateways = append(newNdf.Gateways, newGateway)

	// Update pool with the new Ndf
	hostPool.UpdateNdf(newNdf)

	// Check that the ndf update flag has been set
	if !hostPool.isNdfUpdated {
		t.Errorf("Expected ndf updated flag to be set after updateNdf call")
	}

	// Check that the host pool's ndf has been modified properly
	if !reflect.DeepEqual(newNdf, hostPool.ndf) {
		t.Errorf("Host pool ndf not updated to new ndf.")
	}
}

// Full test
func TestHostPool_GetPreferred(t *testing.T) {
	manager := newHappyManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways)) - 1

	// Pull all gateways from ndf into host manager
	hostMap := make(map[id.ID]bool, 0)
	targets := make([]*id.ID, 0)
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
		}
		// Add mock gateway to manager
		_, err = manager.AddHost(gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Errorf("Could not add mock host to manager: %v", err)
			t.FailNow()
		}

		hostMap[*gwId] = true
		targets = append(targets, gwId)

	}

	// Call the constructor
	testPool, err := NewHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	retrievedList := testPool.GetPreferred(targets)
	if len(retrievedList) != len(targets) {
		t.Errorf("Requested list did not output requested length." +
			"\n\tExpected: %d" +
			"\n\tReceived: %v", len(targets), len(retrievedList))
	}

	// In case where all requested gateways are present
	// ensure requested hosts were returned
	for _, gwID := range targets {
		if !hostMap[*gwID] {
			t.Errorf("A target gateways which should have been returned was not." +
				"\n\tExpected: %v", gwID)
		}
	}

	// Replace a request with a gateway not in pool
	targets[3] = id.NewIdFromUInt(74, id.Gateway, t)
	retrievedList = testPool.GetPreferred(targets)
	if len(retrievedList) != len(targets) {
		t.Errorf("Requested list did not output requested length." +
			"\n\tExpected: %d" +
			"\n\tReceived: %v", len(targets), len(retrievedList))
	}

	// In case where all requested gateways are present
	// ensure requested hosts were returned
	for _, gwID := range targets {
		if !hostMap[*gwID] {
			t.Errorf("A target gateways which should have been returned was not." +
				"\n\tExpected: %v", gwID)
		}
	}

}

// Unit test
func TestHostPool_GetAnyList(t *testing.T) {
	manager := newHappyManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways))

	// Pull all gateways from ndf into host manager
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
		}
		// Add mock gateway to manager
		_, err = manager.AddHost(gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Errorf("Could not add mock host to manager: %v", err)
			t.FailNow()
		}

	}

	// Call the constructor
	testPool, err := NewHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	requested := 3
	anyList := testPool.GetAnyList(requested)
	if len(anyList) != requested {
		t.Errorf("GetAnyList did not get requested length." +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", requested, len(anyList))
	}

	for _, h := range anyList {
		_, ok := manager.GetHost(h.GetId())
		if !ok {
			t.Errorf("Host %s in retrieved list not in manager", h)
		}
	}

	// Request more than are in host list
	largeRequest := requested*1000
	largeRetrieved := testPool.GetAnyList(largeRequest)
	if len(largeRetrieved) != len(testPool.hostList) {
		t.Errorf("Large request should result in a list of all in host list")
	}

}

// Full happy path test
func TestReplaceHost(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:  manager,
		hostList: make([]*connect.Host, newIndex+1),
		hostMap:  make(map[id.ID]uint32),
		ndf:      testNdf,
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
		t.Errorf("Index pulled from map not expected value."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", newIndex, retrievedIndex)
	}

	// Check that the state of the list list been correctly updated
	retrievedHost := hostPool.hostList[newIndex]
	if !gwIdOne.Cmp(retrievedHost.GetId()) {
		t.Errorf("Id pulled from list is not expected."+
			"\n\tExpected: %s"+
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
		t.Errorf("Id pulled from list is not expected."+
			"\n\tExpected: %s"+
			"\n\tReceived: %s", gwIdTwo, retrievedHost.GetId())
	}

	// Check the state of the map has been correctly removed for the old gateway
	retrievedOldIndex, ok := hostPool.hostMap[*gwIdOne]
	if ok {
		t.Errorf("Exoected old gateway to be cleared from map")
	}
	if retrievedOldIndex != 0 {
		t.Errorf("Index pulled from map with old gateway as the key "+
			"was not cleared."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", 0, retrievedOldIndex)
	}

	// Check the state of the map has been correctly updated for the old gateway
	retrievedIndex, ok = hostPool.hostMap[*gwIdTwo]
	if !ok {
		t.Errorf("Expected insertion of gateway ID into map")
	}
	if retrievedIndex != newIndex {
		t.Errorf("Index pulled from map using new gateway as the key "+
			"was not updated."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", newIndex, retrievedIndex)
	}

}

// Error path, could not get host
func TestReplaceHost_Error(t *testing.T) {
	manager := newHappyManager()

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:  manager,
		hostList: make([]*connect.Host, 1),
		hostMap:  make(map[id.ID]uint32),
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
		manager:    manager,
		hostList:   make([]*connect.Host, newIndex+1),
		hostMap:    make(map[id.ID]uint32),
		ndf:        testNdf,
		poolParams: params,
		rng:        rng,
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

	// Construct a host past the error threshold
	errorThresholdIndex := 0
	overThreshold := params.errThreshold + 25
	badMetric := &connect.Metric{ErrCounter: &overThreshold}
	hostList[errorThresholdIndex].SetMetricsTesting(badMetric, t)
	oldHost := hostList[0]

	// Call prune host pool
	err := hostPool.pruneHostPool()
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v", err)
	}

	// Check that the host map has been properly updated
	for _, h := range hostList {
		_, ok := hostPool.hostMap[*h.GetId()]
		if !ok {
			t.Errorf("Gateway %s was not placed in host map after pruning", h.GetId().String())
		}
	}

	// Check that the host list has been has been properly updated
	// at the index with a host past the error threshold
	retrievedHost := hostPool.hostList[errorThresholdIndex]
	if reflect.DeepEqual(oldHost, retrievedHost) {
		t.Errorf("Expected host list to have it's bad host replaced." +
			"Contains old host information after pruning")
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
		manager:    manager,
		hostList:   make([]*connect.Host, newIndex+1),
		hostMap:    make(map[id.ID]uint32),
		ndf:        testNdf,
		poolParams: params,
		rng:        rng,
	}

	// Call prune
	err := hostPool.pruneHostPool()
	if err == nil {
		t.Errorf("Gateways should not be available: " +
			"not enough gateways in ndf compared to param's pool size")
	}

}

// Unit test
func TestAddGateway(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways))

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:        manager,
		hostList:       make([]*connect.Host, newIndex+1),
		hostMap:        make(map[id.ID]uint32),
		ndf:            testNdf,
		addGatewayChan: make(chan network.NodeGateway),
		storage:        storage.InitTestingSession(t),
	}

	ndfIndex := 0

	gwId, err := id.Unmarshal(testNdf.Gateways[ndfIndex].ID)
	if err != nil {
		t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
	}

	hostPool.addGateway(gwId, ndfIndex)

	_, ok := manager.GetHost(gwId)
	if !ok {
		t.Errorf("Unsuccessfully added host to manager")
	}
}

// Unit test
func TestRemoveGateway(t *testing.T) {
	manager := newHappyManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()
	params.poolSize = uint32(len(testNdf.Gateways))

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:        manager,
		hostList:       make([]*connect.Host, newIndex+1),
		hostMap:        make(map[id.ID]uint32),
		ndf:            testNdf,
		addGatewayChan: make(chan network.NodeGateway),
		storage:        storage.InitTestingSession(t),
	}

	ndfIndex := 0

	gwId, err := id.Unmarshal(testNdf.Gateways[ndfIndex].ID)
	if err != nil {
		t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
	}

	// Add the new gateway host
	h, err := hostPool.manager.AddHost(gwId, "", nil, params.hostParams)
	if err != nil {
		jww.ERROR.Printf("Could not add gateway host %s: %+v", gwId, err)
	}

	// Manually add host information
	hostPool.hostMap[*gwId] = uint32(ndfIndex)
	hostPool.hostList[ndfIndex] = h

	// Call the removal
	hostPool.removeGateway(gwId)

	// Check that the map and list have been updated
	if hostPool.hostList[ndfIndex] != nil {
		t.Errorf("Host list index was not set to nil after removal")
	}

	if _, ok := hostPool.hostMap[*gwId]; ok {
		t.Errorf("Host map did not delete host entry")
	}
}
