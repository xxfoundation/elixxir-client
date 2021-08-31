///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package gateway Handles functionality related to providing Gateway connect.Host objects
// for message sending to the rest of the client repo
// Used to minimize # of open connections on mobile clients

package gateway

import (
	"encoding/binary"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"golang.org/x/net/context"
	"io"
	"math"
	"strings"
	"sync"
	"time"
)

// List of errors that initiate a Host replacement
var errorsList = []string{context.DeadlineExceeded.Error(), "connection refused", "host disconnected",
	"transport is closing", "all SubConns are in TransientFailure", "Last try to connect",
	ndf.NO_NDF, "Host is in cool down"}

// HostManager Interface allowing storage and retrieval of Host objects
type HostManager interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error)
	RemoveHost(hid *id.ID)
}

// Filter filters out IDs from the provided map based on criteria in the NDF.
// The passed in map is a map of the NDF for easier acesss.  The map is ID -> index in the NDF
// There is no multithreading, the filter function can either edit the passed map or make a new one
// and return it.  The general pattern is to loop through the map, then look up data about the node
// in the ndf to make a filtering decision, then add them to a new map if they are accepted.
type Filter func(map[id.ID]int, *ndf.NetworkDefinition) map[id.ID]int

// HostPool Handles providing hosts to the Client
type HostPool struct {
	hostMap  map[id.ID]uint32 // map key to its index in the slice
	hostList []*connect.Host  // each index in the slice contains the value
	hostMux  sync.RWMutex     // Mutex for the above map/list combination

	ndfMap map[id.ID]int // map gateway ID to its index in the ndf
	ndf    *ndf.NetworkDefinition
	ndfMux sync.RWMutex

	poolParams     PoolParams
	rng            *fastRNG.StreamGenerator
	storage        *storage.Session
	manager        HostManager
	addGatewayChan chan network.NodeGateway

	filterMux sync.Mutex
	filter    Filter
}

// PoolParams Allows configuration of HostPool parameters
type PoolParams struct {
	MaxPoolSize uint32 // Maximum number of Hosts in the HostPool
	PoolSize    uint32 // Allows override of HostPool size. Set to zero for dynamic size calculation
	// TODO: Move up a layer
	ProxyAttempts uint32             // How many proxies will be used in event of send failure
	HostParams    connect.HostParams // Parameters for the creation of new Host objects
}

// DefaultPoolParams Returns a default set of PoolParams
func DefaultPoolParams() PoolParams {
	p := PoolParams{
		MaxPoolSize:   30,
		ProxyAttempts: 5,
		PoolSize:      0,
		HostParams:    connect.GetDefaultHostParams(),
	}
	p.HostParams.MaxRetries = 1
	p.HostParams.AuthEnabled = false
	p.HostParams.EnableCoolOff = false
	p.HostParams.NumSendsBeforeCoolOff = 1
	p.HostParams.CoolOffTimeout = 5 * time.Minute
	p.HostParams.SendTimeout = 2000 * time.Millisecond
	return p
}

// Build and return new HostPool object
func newHostPool(poolParams PoolParams, rng *fastRNG.StreamGenerator,
	netDef *ndf.NetworkDefinition, getter HostManager, storage *storage.Session,
	addGateway chan network.NodeGateway) (*HostPool, error) {
	var err error

	// Determine size of HostPool
	if poolParams.PoolSize == 0 {
		poolParams.PoolSize, err = getPoolSize(uint32(len(netDef.Gateways)),
			poolParams.MaxPoolSize)
		if err != nil {
			return nil, err
		}
	}

	result := &HostPool{
		manager:        getter,
		hostMap:        make(map[id.ID]uint32),
		hostList:       make([]*connect.Host, poolParams.PoolSize),
		poolParams:     poolParams,
		ndf:            netDef,
		rng:            rng,
		storage:        storage,
		addGatewayChan: addGateway,

		// Initialise the filter so it does not filter any IDs
		filter: func(m map[id.ID]int, _ *ndf.NetworkDefinition) map[id.ID]int {
			return m
		},
	}

	// Propagate the NDF
	err = result.updateConns()
	if err != nil {
		return nil, err
	}

	// Get the last used list of hosts and use it to seed the host pool list
	hostList, err := storage.HostList().Get()
	numHostsAdded := 0
	if err == nil {
		for _, hid := range hostList {
			err := result.replaceHostNoStore(hid, uint32(numHostsAdded))
			if err != nil {
				jww.WARN.Printf("Unable to add stored host %s: %s", hid, err.Error())
			} else {
				numHostsAdded++
			}
		}
	} else {
		jww.WARN.Printf("Building new HostPool because no HostList stored: %+v", err)
	}

	// Build the initial HostPool and return
	for i := numHostsAdded; i < len(result.hostList); i++ {
		err := result.forceReplace(uint32(i))
		if err != nil {
			return nil, err
		}
	}

	jww.INFO.Printf("Initialized HostPool with size: %d/%d", poolParams.PoolSize, len(netDef.Gateways))
	return result, nil
}

// UpdateNdf Mutates internal ndf to the given ndf
func (h *HostPool) UpdateNdf(ndf *ndf.NetworkDefinition) {
	if len(ndf.Gateways) == 0 {
		jww.WARN.Printf("Unable to UpdateNdf: no gateways available")
		return
	}

	h.ndfMux.Lock()
	h.ndf = ndf

	h.hostMux.Lock()
	err := h.updateConns()
	h.hostMux.Unlock()
	if err != nil {
		jww.ERROR.Printf("Unable to updateConns: %+v", err)
	}
	h.ndfMux.Unlock()
}

// SetFilter sets the filter used to filter gateways from the ID map.
func (h *HostPool) SetFilter(f Filter) {
	h.filterMux.Lock()
	defer h.filterMux.Unlock()

	h.filter = f
}

// getFilter returns the filter used to filter gateways from the ID map.
func (h *HostPool) getFilter() Filter {
	h.filterMux.Lock()
	defer h.filterMux.Unlock()

	return h.filter
}

// Obtain a random, unique list of Hosts of the given length from the HostPool
func (h *HostPool) getAny(length uint32, excluded []*id.ID) []*connect.Host {
	if length > h.poolParams.PoolSize {
		length = h.poolParams.PoolSize
	}

	checked := make(map[uint32]interface{}) // Keep track of Hosts already selected to avoid duplicates
	if excluded != nil {
		// Add excluded Hosts to already-checked list
		for i := range excluded {
			gwId := excluded[i]
			if idx, ok := h.hostMap[*gwId]; ok {
				checked[idx] = nil
			}
		}
	}

	result := make([]*connect.Host, 0, length)
	rng := h.rng.GetStream()
	h.hostMux.RLock()
	for i := uint32(0); i < length; {
		// If we've checked the entire HostPool, bail
		if uint32(len(checked)) >= h.poolParams.PoolSize {
			break
		}

		// Check the next HostPool index
		gwIdx := readRangeUint32(0, h.poolParams.PoolSize, rng)
		if _, ok := checked[gwIdx]; !ok {
			result = append(result, h.hostList[gwIdx])
			checked[gwIdx] = nil
			i++
		}
	}
	h.hostMux.RUnlock()
	rng.Close()

	return result
}

// Obtain a specific connect.Host from the manager, irrespective of the HostPool
func (h *HostPool) getSpecific(target *id.ID) (*connect.Host, bool) {
	return h.manager.GetHost(target)
}

// Try to obtain the given targets from the HostPool
// If each is not present, obtain a random replacement from the HostPool
func (h *HostPool) getPreferred(targets []*id.ID) []*connect.Host {
	checked := make(map[uint32]interface{}) // Keep track of Hosts already selected to avoid duplicates
	length := len(targets)
	if length > int(h.poolParams.PoolSize) {
		length = int(h.poolParams.PoolSize)
	}
	result := make([]*connect.Host, length)

	rng := h.rng.GetStream()
	h.hostMux.RLock()
	for i := 0; i < length; {
		if hostIdx, ok := h.hostMap[*targets[i]]; ok {
			result[i] = h.hostList[hostIdx]
			checked[hostIdx] = nil
			i++
			continue
		}

		gwIdx := readRangeUint32(0, h.poolParams.PoolSize, rng)
		if _, ok := checked[gwIdx]; !ok {
			result[i] = h.hostList[gwIdx]
			checked[gwIdx] = nil
			i++
		}
	}
	h.hostMux.RUnlock()
	rng.Close()

	return result
}

// Replaces the given hostId in the HostPool if the given hostErr is in errorList
// Returns whether the host was replaced
func (h *HostPool) checkReplace(hostId *id.ID, hostErr error) (bool, error) {
	var err error
	// Check if Host should be replaced
	doReplace := false
	if hostErr != nil {
		for _, errString := range errorsList {
			if strings.Contains(hostErr.Error(), errString) {
				// Host needs to be replaced, flag and continue
				doReplace = true
				break
			}
		}
	}

	if doReplace {
		// If the Host is still in the pool
		h.hostMux.Lock()
		if oldPoolIndex, ok := h.hostMap[*hostId]; ok {
			// Replace it
			h.ndfMux.RLock()
			err = h.forceReplace(oldPoolIndex)
			h.ndfMux.RUnlock()
		}
		h.hostMux.Unlock()
	}
	return doReplace, err
}

// Replace given Host index with a new, randomly-selected Host from the NDF
func (h *HostPool) forceReplace(oldPoolIndex uint32) error {
	rng := h.rng.GetStream()
	defer rng.Close()

	// Loop until a replacement Host is found
	for {
		// Randomly select a new Gw by index in the NDF
		ndfIdx := readRangeUint32(0, uint32(len(h.ndf.Gateways)), rng)
		jww.DEBUG.Printf("Attempting to replace Host at HostPool %d with Host at NDF %d...", oldPoolIndex, ndfIdx)

		// Use the random ndfIdx to obtain a GwId from the NDF
		gwId, err := id.Unmarshal(h.ndf.Gateways[ndfIdx].ID)
		if err != nil {
			return errors.WithMessage(err, "failed to get Gateway for pruning")
		}

		// Verify the Gateway's Node is not Stale before adding to HostPool
		nodeId := gwId.DeepCopy()
		nodeId.SetType(id.Node)
		nodeNdfIdx := h.ndfMap[*nodeId]
		isNodeStale := h.ndf.Nodes[nodeNdfIdx].Status != ndf.Stale
		if isNodeStale {
			continue
		}

		// Verify the new GwId is not already in the hostMap
		if _, ok := h.hostMap[*gwId]; !ok {
			// If it is a new GwId, replace the old Host with the new Host
			return h.replaceHost(gwId, oldPoolIndex)
		}
	}
}

// replaceHost replaces the given slot in the HostPool with a new Gateway with
// the specified ID. The resulting host list is saved to storage.
func (h *HostPool) replaceHost(newId *id.ID, oldPoolIndex uint32) error {
	err := h.replaceHostNoStore(newId, oldPoolIndex)
	if err != nil {
		return err
	}

	// Convert list of of non-nil and non-zero hosts to ID list
	idList := make([]*id.ID, 0, len(h.hostList))
	for _, host := range h.hostList {
		if host.GetId() != nil && !host.GetId().Cmp(&id.ID{}) {
			idList = append(idList, host.GetId())
		}
	}

	// Save the list to storage
	return h.storage.HostList().Store(idList)
}

// replaceHostNoStore replaces the given slot in the HostPool with a new Gateway
// with the specified ID.
func (h *HostPool) replaceHostNoStore(newId *id.ID, oldPoolIndex uint32) error {
	// Obtain that GwId's Host object
	newHost, ok := h.manager.GetHost(newId)
	if !ok {
		return errors.Errorf("host for gateway %s could not be "+
			"retrieved", newId)
	}

	// Keep track of oldHost for cleanup
	oldHost := h.hostList[oldPoolIndex]

	// Use the poolIdx to overwrite the random Host in the corresponding index
	// in the hostList
	h.hostList[oldPoolIndex] = newHost
	// Use the GwId to keep track of the new random Host's index in the hostList
	h.hostMap[*newId] = oldPoolIndex

	// Clean up and move onto next Host
	if oldHost != nil {
		delete(h.hostMap, *oldHost.GetId())
		go oldHost.Disconnect()
	}
	jww.DEBUG.Printf("Replaced Host at %d with new Host %s", oldPoolIndex,
		newId.String())

	return nil
}

// Force-add the Gateways to the HostPool, each replacing a random Gateway
func (h *HostPool) forceAdd(gwId *id.ID) error {
	rng := h.rng.GetStream()
	h.hostMux.Lock()
	defer h.hostMux.Unlock()
	defer rng.Close()

	// Verify the GwId is not already in the hostMap
	if _, ok := h.hostMap[*gwId]; ok {
		// If it is, skip
		return nil
	}

	// Randomly select another Gateway in the HostPool for replacement
	poolIdx := readRangeUint32(0, h.poolParams.PoolSize, rng)
	return h.replaceHost(gwId, poolIdx)
}

// Updates the internal HostPool with any changes to the NDF
func (h *HostPool) updateConns() error {
	// Prepare NDFs for comparison
	newMap, err := convertNdfToMap(h.ndf)
	if err != nil {
		return errors.Errorf("Unable to convert new NDF to set: %+v", err)
	}

	// Filter out gateway IDs
	newMap = h.getFilter()(newMap, h.ndf)

	// Handle adding Gateways
	for gwId, ndfIdx := range newMap {
		if _, ok := h.ndfMap[gwId]; !ok {
			// If GwId in newMap is not in ndfMap, add the Gateway
			h.addGateway(gwId.DeepCopy(), ndfIdx)
		}
	}

	// Handle removing Gateways
	for gwId := range h.ndfMap {
		if _, ok := newMap[gwId]; !ok {
			// If GwId in ndfMap is not in newMap, remove the Gateway
			h.removeGateway(gwId.DeepCopy())
		}
	}

	// Update the internal NDF set
	h.ndfMap = newMap
	return nil
}

// Takes ndf.Gateways and puts their IDs into a map object
func convertNdfToMap(ndf *ndf.NetworkDefinition) (map[id.ID]int, error) {
	result := make(map[id.ID]int)
	if ndf == nil {
		return result, nil
	}

	// Process Node and Gateway Ids into set
	// NOTE: We expect len(ndf.Gateways) == len(ndf.Nodes)
	for i := range ndf.Gateways {
		gw := ndf.Gateways[i]
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			return nil, err
		}
		result[*gwId] = i

		node := ndf.Nodes[i]
		nodeId, err := id.Unmarshal(node.ID)
		if err != nil {
			return nil, err
		}
		result[*nodeId] = i
	}

	return result, nil
}

// updateConns helper for removing old Gateways
func (h *HostPool) removeGateway(gwId *id.ID) {
	h.manager.RemoveHost(gwId)
	// If needed, replace the removed Gateway in the HostPool with a new one
	if poolIndex, ok := h.hostMap[*gwId]; ok {
		err := h.forceReplace(poolIndex)
		if err != nil {
			jww.ERROR.Printf("Unable to removeGateway: %+v", err)
		}
	}
}

// updateConns helper for adding new Gateways
func (h *HostPool) addGateway(gwId *id.ID, ndfIndex int) {
	gw := h.ndf.Gateways[ndfIndex]

	// Check if the host exists
	host, ok := h.manager.GetHost(gwId)
	if !ok {

		// Check if gateway ID collides with an existing hard coded ID
		if id.CollidesWithHardCodedID(gwId) {
			jww.ERROR.Printf("Gateway ID invalid, collides with a "+
				"hard coded ID. Invalid ID: %v", gwId.Marshal())
		}

		// Add the new gateway host
		_, err := h.manager.AddHost(gwId, gw.Address, []byte(gw.TlsCertificate), h.poolParams.HostParams)
		if err != nil {
			jww.ERROR.Printf("Could not add gateway host %s: %+v", gwId, err)
		}

		// Send AddGateway event if we do not already possess keys for the GW
		if !h.storage.Cmix().Has(gwId) {
			ng := network.NodeGateway{
				Node:    h.ndf.Nodes[ndfIndex],
				Gateway: gw,
			}

			select {
			case h.addGatewayChan <- ng:
			default:
				jww.WARN.Printf("Unable to send AddGateway event for id %s", gwId.String())
			}
		}

	} else if host.GetAddress() != gw.Address {
		host.UpdateAddress(gw.Address)
	}
}

// getPoolSize determines the size of the HostPool based on the size of the NDF
func getPoolSize(ndfLen, maxSize uint32) (uint32, error) {
	// Verify the NDF has at least one Gateway for the HostPool
	if ndfLen == 0 {
		return 0, errors.Errorf("Unable to create HostPool: no gateways available")
	}

	// PoolSize = ceil(sqrt(len(ndf,Gateways)))
	poolSize := uint32(math.Ceil(math.Sqrt(float64(ndfLen))))
	if poolSize > maxSize {
		return maxSize, nil
	}
	return poolSize, nil
}

// readUint32 reads an integer from an io.Reader (which should be a CSPRNG)
func readUint32(rng io.Reader) uint32 {
	var rndBytes [4]byte
	i, err := rng.Read(rndBytes[:])
	if i != 4 || err != nil {
		jww.FATAL.Panicf("cannot read from rng: %+v", err)
	}
	return binary.BigEndian.Uint32(rndBytes[:])
}

// readRangeUint32 reduces an integer from 0, MaxUint32 to the range start, end
func readRangeUint32(start, end uint32, rng io.Reader) uint32 {
	size := end - start
	// note we could just do the part inside the () here, but then extra
	// can == size which means a little bit of range is wastes, either
	// choice seems negligible so we went with the "more correct"
	extra := (math.MaxUint32%size + 1) % size
	limit := math.MaxUint32 - extra
	// Loop until we read something inside the limit
	for {
		res := readUint32(rng)
		if res > limit {
			continue
		}
		return (res % size) + start
	}
}
