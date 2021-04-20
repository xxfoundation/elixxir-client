///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"fmt"
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
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	_, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}
}

// Unit test
func TestHostPool_ManageHostPool(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)

	// Construct custom params
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	// Construct a list of new gateways/nodes to add to ndf
	newGatewayLen := len(testNdf.Gateways)
	newGateways := make([]ndf.Gateway, newGatewayLen)
	newNodes := make([]ndf.Node, newGatewayLen)
	for i := 0; i < newGatewayLen; i++ {
		// Construct gateways
		gwId := id.NewIdFromUInt(uint64(100+i), id.Gateway, t)
		newGateways[i] = ndf.Gateway{ID: gwId.Bytes()}
		// Construct nodes
		nodeId := gwId.DeepCopy()
		nodeId.SetType(id.Node)
		newNodes[i] = ndf.Node{ID: nodeId.Bytes()}

	}

	newNdf := getTestNdf(t)
	// Update the ndf, removing some gateways at a cutoff
	newNdf.Gateways = newGateways
	newNdf.Nodes = newNodes

	testPool.UpdateNdf(newNdf)

	// Check that old gateways are not in pool
	for _, ndfGw := range testNdf.Gateways {
		gwId, err := id.Unmarshal(ndfGw.ID)
		if err != nil {
			t.Errorf("Failed to marshal gateway id for %v", ndfGw)
		}
		if _, ok := testPool.hostMap[*gwId]; ok {
			t.Errorf("Expected gateway %v to be removed from pool", gwId)
		}
	}
}

// Full happy path test
func TestHostPool_ReplaceHost(t *testing.T) {
	manager := newMockManager()
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
func TestHostPool_ReplaceHost_Error(t *testing.T) {
	manager := newMockManager()

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

// Unit test
func TestHostPool_ForceReplace(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)

	// Construct custom params
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	// Add all gateways to hostPool's map
	for index, gw := range testNdf.Gateways {
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Fatalf("Failed to unmarshal ID in mock ndf: %v", err)
		}

		err = testPool.replaceHost(gwId, uint32(index))
		if err != nil {
			t.Fatalf("Failed to replace host in set-up: %v", err)
		}
	}

	oldGatewayIndex := 0
	oldHost := testPool.hostList[oldGatewayIndex]

	// Force replace the gateway at a given index
	err = testPool.forceReplace(uint32(oldGatewayIndex))
	if err != nil {
		t.Errorf("Failed to force replace: %v", err)
	}

	// Ensure that old gateway has been removed from the map
	if _, ok := testPool.hostMap[*oldHost.GetId()]; ok {
		t.Errorf("Expected old host to be removed from map")
	}

	// Ensure we are disconnected from the old host
	if isConnected, _ := oldHost.Connected(); isConnected {
		t.Errorf("Failed to disconnect from old host %s", oldHost)
	}

}

// Unit test
func TestHostPool_CheckReplace(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)

	// Construct custom params
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways)) - 5

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	// Call check replace
	oldGatewayIndex := 0
	oldHost := testPool.hostList[oldGatewayIndex]
	expectedError := fmt.Errorf(errorsList[0])
	err = testPool.checkReplace(oldHost.GetId(), expectedError)
	if err != nil {
		t.Errorf("Failed to check replace: %v", err)
	}

	// Ensure that old gateway has been removed from the map
	if _, ok := testPool.hostMap[*oldHost.GetId()]; ok {
		t.Errorf("Expected old host to be removed from map")
	}

	// Ensure we are disconnected from the old host
	if isConnected, _ := oldHost.Connected(); isConnected {
		t.Errorf("Failed to disconnect from old host %s", oldHost)
	}

	// Check that an error not in the global list results in a no-op
	goodGatewayIndex := 0
	goodGateway := testPool.hostList[goodGatewayIndex]
	unexpectedErr := fmt.Errorf("Not in global error list")
	err = testPool.checkReplace(oldHost.GetId(), unexpectedErr)
	if err != nil {
		t.Errorf("Failed to check replace: %v", err)
	}

	// Ensure that gateway with an unexpected error was not modified
	if _, ok := testPool.hostMap[*goodGateway.GetId()]; !ok {
		t.Errorf("Expected gateway with non-expected error to not be modified")
	}

	// Ensure gateway host has not been disconnected
	if isConnected, _ := oldHost.Connected(); isConnected {
		t.Errorf("Should not disconnect from  %s", oldHost)
	}

}

// TODO: Adapt to new methods
// Happy path
//func TestHostPool_PruneHostPool(t *testing.T) {
//	manager := newMockManager()
//	testNdf := getTestNdf(t)
//	newIndex := uint32(20)
//	params := DefaultPoolParams()
//	params.PoolSize = uint32(len(testNdf.Gateways))
//	rng := csprng.NewSystemRNG()
//
//	// Construct a manager (bypass business logic in constructor)
//	hostPool := &HostPool{
//		manager:    manager,
//		hostList:   make([]*connect.Host, newIndex+1),
//		hostMap:    make(map[id.ID]uint32),
//		ndf:        testNdf,
//		poolParams: params,
//		rng:        rng,
//	}
//
//	// Pull all gateways from ndf into host manager
//	hostList := make([]*connect.Host, 0)
//	for _, gw := range testNdf.Gateways {
//
//		gwId, err := id.Unmarshal(gw.ID)
//		if err != nil {
//			t.Errorf("Failed to unmarshal ID in mock ndf: %v", err)
//		}
//		// Add mock gateway to manager
//		h, err := manager.AddHost(gwId, "", nil, connect.GetDefaultHostParams())
//		if err != nil {
//			t.Errorf("Could not add mock host to manager: %v", err)
//			t.FailNow()
//		}
//
//		hostList = append(hostList, h)
//
//	}
//
//	// Construct a host past the error threshold
//	errorThresholdIndex := 0
//	overThreshold := params.ErrThreshold + 25
//	hostList[errorThresholdIndex].SetMetricsTesting(connect.NewMetricTesting(overThreshold, t), t)
//	oldHost := hostList[0]
//
//	// Call prune host pool
//	err := hostPool.pruneHostPool()
//	if err != nil {
//		t.Errorf("Unexpected error in happy path: %v", err)
//	}
//
//	// Check that the host map has been properly updated
//	for _, h := range hostList {
//		_, ok := hostPool.hostMap[*h.GetId()]
//		if !ok {
//			t.Errorf("Gateway %s was not placed in host map after pruning", h.GetId().String())
//		}
//	}
//
//	// Check that the host list has been has been properly updated
//	// at the index with a host past the error threshold
//	retrievedHost := hostPool.hostList[errorThresholdIndex]
//	if reflect.DeepEqual(oldHost, retrievedHost) {
//		t.Errorf("Expected host list to have it's bad host replaced. " +
//			"Contains old host information after pruning")
//	}
//
//}
//
//// Error path: not enough gateways in ndf compared to
//// required pool size
//func TestHostPool_PruneHostPool_Error(t *testing.T) {
//	manager := newMockManager()
//	testNdf := getTestNdf(t)
//	newIndex := uint32(20)
//	params := DefaultPoolParams()
//
//	// Trigger the case where the Ndf doesn't have enough gateways
//	params.PoolSize = uint32(len(testNdf.Gateways)) + 1
//	rng := csprng.NewSystemRNG()
//
//	// Construct a manager (bypass business logic in constructor)
//	hostPool := &HostPool{
//		manager:    manager,
//		hostList:   make([]*connect.Host, newIndex+1),
//		hostMap:    make(map[id.ID]uint32),
//		ndf:        testNdf,
//		poolParams: params,
//		rng:        rng,
//	}
//
//	// Call prune
//	err := hostPool.pruneHostPool()
//	if err == nil {
//		t.Errorf("Gateways should not be available: " +
//			"not enough gateways in ndf compared to param's pool size")
//	}
//
//}

// Unit test
func TestHostPool_UpdateNdf(t *testing.T) {
	manager := newMockManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)

	// Construct a manager (bypass business logic in constructor)
	hostPool := &HostPool{
		manager:  manager,
		hostList: make([]*connect.Host, newIndex+1),
		hostMap:  make(map[id.ID]uint32),
		ndf:      testNdf,
		storage:  storage.InitTestingSession(t),
	}

	// Construct a new Ndf different from original one above
	newNdf := getTestNdf(t)
	newGateway := ndf.Gateway{
		ID: id.NewIdFromUInt(27, id.Gateway, t).Bytes(),
	}
	newNode := ndf.Node{
		ID: id.NewIdFromUInt(27, id.Node, t).Bytes(),
	}
	newNdf.Gateways = append(newNdf.Gateways, newGateway)
	newNdf.Nodes = append(newNdf.Nodes, newNode)

	// Update pool with the new Ndf
	hostPool.UpdateNdf(newNdf)

	// Check that the host pool's ndf has been modified properly
	if !reflect.DeepEqual(newNdf, hostPool.ndf) {
		t.Errorf("Host pool ndf not updated to new ndf.")
	}
}

// Full test
func TestHostPool_GetPreferred(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	retrievedList := testPool.getPreferred(targets)
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
	retrievedList = testPool.getPreferred(targets)
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

// Unit test
func TestHostPool_GetAny(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	requested := 3
	anyList := testPool.getAny(uint32(requested), nil)
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
	largeRetrieved := testPool.getAny(largeRequest, nil)
	if len(largeRetrieved) != len(testPool.hostList) {
		t.Errorf("Large request should result in a list of all in host list")
	}

}

// Unit test
func TestHostPool_ForceAdd(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	// Construct a list of new gateways to add
	newGatewayLen := 10
	newGateways := make([]*id.ID, newGatewayLen)
	for i := 0; i < newGatewayLen; i++ {
		gwId := id.NewIdFromUInt(uint64(100+i), id.Gateway, t)
		// Add mock gateway to manager
		_, err = manager.AddHost(gwId, "", nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not add mock host to manager: %v", err)
		}
		newGateways[i] = gwId
	}

	// forceAdd list of gateways
	err = testPool.forceAdd(newGateways)
	if err != nil {
		t.Errorf("Could not add gateways: %v", err)
	}

	// check that gateways have been added to the map
	for _, gw := range newGateways {
		if _, ok := testPool.hostMap[*gw]; !ok {
			t.Errorf("Failed to forcefully add new gateway ID: %v", gw)
		}
	}

}

// Unit test which only adds information to ndf
func TestHostPool_UpdateConns_AddGateways(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	// Construct a list of new gateways/nodes to add to ndf
	newGatewayLen := 10
	newGateways := make([]ndf.Gateway, newGatewayLen)
	newNodes := make([]ndf.Node, newGatewayLen)
	for i := 0; i < newGatewayLen; i++ {
		// Construct gateways
		gwId := id.NewIdFromUInt(uint64(100+i), id.Gateway, t)
		newGateways[i] = ndf.Gateway{ID: gwId.Bytes()}
		// Construct nodes
		nodeId := gwId.DeepCopy()
		nodeId.SetType(id.Node)
		newNodes[i] = ndf.Node{ID: nodeId.Bytes()}

	}

	// Update the ndf
	newNdf := getTestNdf(t)
	newNdf.Gateways = append(newNdf.Gateways, newGateways...)
	newNdf.Nodes = append(newNdf.Nodes, newNodes...)

	testPool.UpdateNdf(newNdf)

	// Update the connections
	err = testPool.updateConns()
	if err != nil {
		t.Errorf("Failed to update connections: %v", err)
	}

	// Check that new gateways are in manager
	for _, ndfGw := range newGateways {
		gwId, err := id.Unmarshal(ndfGw.ID)
		if err != nil {
			t.Errorf("Failed to marshal gateway id for %v", ndfGw)
		}
		_, ok := testPool.getSpecific(gwId)
		if !ok {
			t.Errorf("Failed to find gateway %v in manager", gwId)
		}
	}

}

// Unit test which only adds information to ndf
func TestHostPool_UpdateConns_RemoveGateways(t *testing.T) {
	manager := newMockManager()
	rng := csprng.NewSystemRNG()
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	testPool, err := newHostPool(params, rng, testNdf, manager,
		testStorage, addGwChan)
	if err != nil {
		t.Errorf("Failed to create mock host pool: %v", err)
	}

	// Construct a list of new gateways/nodes to add to ndf
	newGatewayLen := len(testNdf.Gateways)
	newGateways := make([]ndf.Gateway, newGatewayLen)
	newNodes := make([]ndf.Node, newGatewayLen)
	for i := 0; i < newGatewayLen; i++ {
		// Construct gateways
		gwId := id.NewIdFromUInt(uint64(100+i), id.Gateway, t)
		newGateways[i] = ndf.Gateway{ID: gwId.Bytes()}
		// Construct nodes
		nodeId := gwId.DeepCopy()
		nodeId.SetType(id.Node)
		newNodes[i] = ndf.Node{ID: nodeId.Bytes()}

	}

	// Update the ndf, replacing old data entirely
	newNdf := getTestNdf(t)
	newNdf.Gateways = newGateways
	newNdf.Nodes = newNodes

	testPool.UpdateNdf(newNdf)

	// Update the connections
	err = testPool.updateConns()
	if err != nil {
		t.Errorf("Failed to update connections: %v", err)
	}

	// Check that old gateways are not in pool
	for _, ndfGw := range testNdf.Gateways {
		gwId, err := id.Unmarshal(ndfGw.ID)
		if err != nil {
			t.Errorf("Failed to marshal gateway id for %v", ndfGw)
		}
		if _, ok := testPool.hostMap[*gwId]; ok {
			t.Errorf("Expected gateway %v to be removed from pool", gwId)
		}
	}
}

// Unit test
func TestHostPool_AddGateway(t *testing.T) {
	manager := newMockManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
func TestHostPool_RemoveGateway(t *testing.T) {
	manager := newMockManager()
	testNdf := getTestNdf(t)
	newIndex := uint32(20)
	params := DefaultPoolParams()
	params.PoolSize = uint32(len(testNdf.Gateways))

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
	h, err := hostPool.manager.AddHost(gwId, "", nil, params.HostParams)
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
