////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
	"time"

	"encoding/json"
	"github.com/golang-collections/collections/set"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

// Unit test
func Test_newHostPool(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	params := DefaultParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	// Pull all gateways from NDF into host manager
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		_, err = manager.AddHost(gwId, "", nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not Add mock host to manager: %+v", err)
		}

	}

	// Call the constructor
	_, err := newHostPool(params, rng, testNdf, manager,
		testStorage, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}
}

// Tests that the hosts are loaded from storage, if they exist.
func Test_newHostPool_HostListStore(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway, len(testNdf.Gateways))
	params := DefaultPoolParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	addedIDs := []*id.ID{
		id.NewIdFromString("testID0", id.Gateway, t),
		id.NewIdFromString("testID1", id.Gateway, t),
		id.NewIdFromString("testID2", id.Gateway, t),
		id.NewIdFromString("testID3", id.Gateway, t),
	}
	err := saveHostList(testStorage.GetKV(), addedIDs)
	if err != nil {
		t.Fatalf("Failed to store host list: %+v", err)
	}

	for i, hid := range addedIDs {
		testNdf.Gateways[i].ID = hid.Marshal()
	}

	// Call the constructor
	mccc := &mockCertCheckerComm{}
	hp, err := newHostPool(params, rng, testNdf, manager, testStorage, addGwChan, mccc)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	// Check that the host list was saved to storage
	hostList, err := getHostList(hp.kv)
	if err != nil {
		t.Errorf("Failed to get host list: %+v", err)
	}

	if !reflect.DeepEqual(addedIDs, hostList) {
		t.Errorf("Failed to save expected host list to storage."+
			"\nexpected: %+v\nreceived: %+v", addedIDs, hostList)
	}
}

func TestPrint(t *testing.T) {
	p := pool{
		hostMap: make(map[id.ID]uint),
	}

	for i := uint(0); i < 5; i++ {
		p.hostMap[*id.NewIdFromUInt(uint64(i), id.Gateway, t)] = i

	}

	data, err := json.Marshal(p.hostMap)
	if err != nil {
		t.Fatalf("Failed to marshal map: %+v", err)
	}

	t.Logf("%s", string(data))

}

// Unit test.
func TestHostPool_ManageHostPool(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway, len(testNdf.Gateways))

	// Construct custom params
	params := DefaultPoolParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	// Pull all gateways from NDF into host manager
	for _, gw := range testNdf.Gateways {

		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			t.Errorf("Failed to unmarshal ID in mock NDF: %+v", err)
		}
		// Add mock gateway to manager
		_, err = manager.AddHost(
			gwId, gw.Address, nil, connect.GetDefaultHostParams())
		if err != nil {
			t.Fatalf("Could not Add mock host to manager: %+v", err)
		}

	}

	// Call the constructor
	mccc := &mockCertCheckerComm{}
	testPool, err := newHostPool(
		params, rng, testNdf, manager, testStorage, addGwChan, mccc)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %+v", err)
	}

	// Construct a list of new gateways/nodes to Add to the NDF
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
		newNodes[i] = ndf.Node{ID: nodeId.Bytes(), Status: ndf.Active}
	}

	// Update the NDF, removing some gateways at a cutoff
	newNdf := getTestNdf(t)
	newNdf.Gateways = newGateways
	newNdf.Nodes = newNodes

	testPool.UpdateNdf(newNdf)

	// Check that old gateways are not in pool
	for _, ndfGw := range testNdf.Gateways {
		gwId, err := id.Unmarshal(ndfGw.ID)
		if err != nil {
			t.Fatalf("Failed to marshal gateway ID for %v", ndfGw)
		}
		if _, ok := testPool.writePool.hostMap[*gwId]; ok {
			t.Errorf("Expected gateway %v to be removed from pool", gwId)
		}
	}
}

// Unit test.
func TestHostPool_UpdateNdf(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway, 150)
	params := DefaultPoolParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	addedIDs := []*id.ID{
		id.NewIdFromString("testID0", id.Gateway, t),
		id.NewIdFromString("testID1", id.Gateway, t),
		id.NewIdFromString("testID2", id.Gateway, t),
		id.NewIdFromString("testID3", id.Gateway, t),
	}
	err := saveHostList(testStorage.GetKV(), addedIDs)
	if err != nil {
		t.Fatalf("Failed to store host list: %+v", err)
	}

	for i, hid := range addedIDs {
		testNdf.Gateways[i].ID = hid.Marshal()
	}

	// Call the constructor
	mccc := &mockCertCheckerComm{}
	testPool, err := newHostPool(params, rng, testNdf, manager, testStorage, addGwChan, mccc)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	stop := stoppable.NewSingle("tester")
	go testPool.runner(stop)
	defer func() {
		if err = stop.Close(); err != nil {
			t.Fatalf("Failed to close stoppable: %+v", err)
		}
	}()

	// Construct a new Ndf different from original one above
	newNdf := getTestNdf(t)
	newGateway := ndf.Gateway{
		ID:      id.NewIdFromUInt(27, id.Gateway, t).Bytes(),
		Address: "0.0.0.3:11420",
	}
	newNode := ndf.Node{
		ID:      id.NewIdFromUInt(27, id.Node, t).Bytes(),
		Address: "0.0.0.3:11420",
	}
	newNdf.Gateways = append(newNdf.Gateways, newGateway)
	newNdf.Nodes = append(newNdf.Nodes, newNode)

	// Update pool with the new Ndf
	testPool.UpdateNdf(newNdf)

	c := make(chan struct{})
	go func() {
		for len(newNdf.Nodes) != len(testPool.ndf.Nodes) ||
			len(newNdf.Gateways) != len(testPool.ndf.Gateways) ||
			len(newNdf.Gateways) != len(testPool.ndfMap) {
			time.Sleep(50 * time.Millisecond)
		}
		c <- struct{}{}
	}()

	select {
	case <-c:
	case <-time.After(5 * time.Second):
		t.Errorf("Timed out waiting for NDF to update. " +
			"Host pool NDF not updated to new NDF.")
	}
}

func TestHostPool_UpdateNdf_AddFilter(t *testing.T) {
	manager := newMockManager()
	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	testNdf := getTestNdf(t)
	testStorage := storage.InitTestingSession(t)
	addGwChan := make(chan network.NodeGateway, 150)
	params := DefaultPoolParams()
	params.MaxPoolSize = uint32(len(testNdf.Gateways))

	addedIDs := []*id.ID{
		id.NewIdFromString("testID0", id.Gateway, t),
		id.NewIdFromString("testID1", id.Gateway, t),
		id.NewIdFromString("testID2", id.Gateway, t),
		id.NewIdFromString("testID3", id.Gateway, t),
	}
	err := saveHostList(testStorage.GetKV(), addedIDs)
	if err != nil {
		t.Fatalf("Failed to store host list: %+v", err)
	}

	for i, hid := range addedIDs {
		testNdf.Gateways[i].ID = hid.Marshal()
	}

	// Call the constructor
	mccc := &mockCertCheckerComm{}
	doneCh := make(chan bool, 1)
	allowedIds := set.New()
	allowedId := id.NewIdFromUInt(27, id.Gateway, t)
	allowedIds.Insert(allowedId.String())
	whitelist := []string{allowedId.String()}

	f := GatewayWhitelistFilter(whitelist)
	params.GatewayFilter = func(unfiltered map[id.ID]int, ndf *ndf.NetworkDefinition) map[id.ID]int {
		doneCh <- true
		filtered := f(unfiltered, ndf)
		return filtered
	}
	testPool, err := newHostPool(params, rng, testNdf, manager, testStorage, addGwChan, mccc)
	if err != nil {
		t.Fatalf("Failed to create mock host pool: %v", err)
	}

	stop := stoppable.NewSingle("tester")
	go testPool.runner(stop)
	defer func() {
		if err = stop.Close(); err != nil {
			t.Fatalf("Failed to close stoppable: %+v", err)
		}
	}()

	// Construct a new Ndf different from original one above
	newNdf := getTestNdf(t)
	newGateway := ndf.Gateway{
		ID:      allowedId.Bytes(),
		Address: "0.0.0.3:11420",
	}
	newNode := ndf.Node{
		ID:      id.NewIdFromUInt(27, id.Node, t).Bytes(),
		Address: "0.0.0.3:11420",
		Status:  ndf.Active,
	}
	newNdf.Gateways = append(newNdf.Gateways, newGateway)
	newNdf.Nodes = append(newNdf.Nodes, newNode)

	select {
	case <-time.After(2 * time.Second):
		t.Fatalf("Did not run filter before timeout")
	case <-doneCh:
	}

	// Update pool with the new Ndf
	testPool.UpdateNdf(newNdf)

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("Did not run filter before timeout")
	case <-doneCh:
	}
	time.Sleep(time.Second)

	// Check that the host pool's NDF has been modified properly
	require.Equal(t, len(newNdf.Nodes), len(testPool.ndf.Nodes))
	require.Equal(t, len(newNdf.Gateways), len(testPool.ndf.Gateways))
	require.Equal(t, allowedIds.Len(), len(testPool.ndfMap))

	for gwID := range testPool.ndfMap {
		if !allowedIds.Has(gwID.String()) {
			t.Fatalf("id in NDF map not in allowed IDs")
		}
	}

	done := false
	testCount := 0
	for !done {
		select {
		case <-testPool.testNodes:
			testCount++
		case <-time.After(250 * time.Millisecond):
			done = true
		}
	}
	if testCount != 1 {
		t.Fatalf("Did not receive expected test count."+
			"\nexpected: %d\nreceived: %d", 1, testCount)
	}
}

type mockCertCheckerComm struct{}

func (mccc *mockCertCheckerComm) GetGatewayTLSCertificate(*connect.Host,
	*pb.RequestGatewayCert) (*pb.GatewayCertificate, error) {
	return &pb.GatewayCertificate{}, nil
}
