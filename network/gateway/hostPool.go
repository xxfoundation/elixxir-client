///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles functionality related to providing Gateway connect.Host objects
// for message sending to the rest of the client repo

package gateway

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
	"math"
	"sync"
	"time"
)

// Interface allowing storage and retrieval of Host objects
type HostManager interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error)
	RemoveHost(hid *id.ID)
}

// Handles providing hosts to the Client
type HostPool struct {
	hostMap  map[id.ID]uint32 // map key to its index in the slice
	hostList []*connect.Host  // each index in the slice contains the value
	hostMux  sync.RWMutex     // Mutex for the above map/list combination

	ndfMap       map[id.ID]int // map gateway ID to its index in the ndf
	ndf          *ndf.NetworkDefinition
	isNdfUpdated bool // indicates the NDF has been updated and needs processed
	ndfMux       sync.RWMutex

	poolParams     PoolParams
	rng            io.Reader
	storage        *storage.Session
	manager        HostManager
	addGatewayChan chan network.NodeGateway
}

// Allows configuration of HostPool parameters
type PoolParams struct {
	poolSize      uint32             // Quantity of Hosts in the HostPool
	errThreshold  uint64             // How many errors will cause a Host to be ejected from the HostPool
	pruneInterval time.Duration      // How frequently the HostPool updates the pool
	hostParams    connect.HostParams // Parameters for the creation of new Host objects
}

// Returns a default set of PoolParams
func DefaultPoolParams() PoolParams {
	return PoolParams{
		poolSize:      30,
		errThreshold:  1,
		pruneInterval: 10 * time.Second,
		hostParams:    connect.GetDefaultHostParams(),
	}
}

// Build and return new HostPool object
func NewHostPool(poolParams PoolParams, rng io.Reader, ndf *ndf.NetworkDefinition, getter HostManager,
	storage *storage.Session, addGateway chan network.NodeGateway) (*HostPool, error) {
	result := &HostPool{
		manager:        getter,
		hostMap:        make(map[id.ID]uint32),
		hostList:       make([]*connect.Host, poolParams.poolSize),
		poolParams:     poolParams,
		ndf:            ndf,
		rng:            rng,
		storage:        storage,
		addGatewayChan: addGateway,
	}

	// Build the initial HostPool
	err := result.pruneHostPool()
	if err != nil {
		return nil, err
	}

	// Convert the NDF into a map object for future comparison
	result.ndfMap, err = convertNdfToMap(ndf)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Mutates internal ndf to the given ndf
func (h *HostPool) UpdateNdf(ndf *ndf.NetworkDefinition) {
	h.ndfMux.Lock()
	h.isNdfUpdated = true
	h.ndf = ndf
	h.ndfMux.Unlock()
}

// Obtain a random, unique list of Hosts of the given length from the HostPool
func (h *HostPool) GetAny(length int) []*connect.Host {
	checked := make(map[uint32]interface{}) // Keep track of Hosts already selected to avoid duplicates
	if length > int(h.poolParams.poolSize) {
		length = int(h.poolParams.poolSize)
	}
	result := make([]*connect.Host, length)
	h.hostMux.RLock()
	for i := 0; i < length; {
		gwIdx := readRangeUint32(0, h.poolParams.poolSize, h.rng)
		if _, ok := checked[gwIdx]; !ok {
			result[i] = h.hostList[gwIdx]
			checked[gwIdx] = nil
			i++
		}
	}
	h.hostMux.RUnlock()

	return result
}

// Obtain a specific connect.Host from the manager, irrespective of the HostPool
func (h *HostPool) GetSpecific(target *id.ID) (*connect.Host, bool) {
	return h.manager.GetHost(target)
}

// Try to obtain the given targets from the HostPool
// If each is not present, obtain a random replacement from the HostPool
func (h *HostPool) GetPreferred(targets []*id.ID) []*connect.Host {
	checked := make(map[uint32]interface{}) // Keep track of Hosts already selected to avoid duplicates
	length := len(targets)
	if length > int(h.poolParams.poolSize) {
		length = int(h.poolParams.poolSize)
	}
	result := make([]*connect.Host, length)

	h.hostMux.RLock()
	for i := 0; i < length; {
		if hostIdx, ok := h.hostMap[*targets[i]]; ok {
			result[i] = h.hostList[hostIdx]
			i++
			continue
		}

		gwIdx := readRangeUint32(0, h.poolParams.poolSize, h.rng)
		if _, ok := checked[gwIdx]; !ok {
			result[i] = h.hostList[gwIdx]
			checked[gwIdx] = nil
			i++
		}
	}
	h.hostMux.RUnlock()

	return result
}

// Start long-running thread and return the thread controller to the caller
func (h *HostPool) StartHostPool() stoppable.Stoppable {
	stopper := stoppable.NewSingle("HostPool")
	jww.INFO.Printf("Starting Host Pool...")
	go h.manageHostPool(stopper)
	return stopper
}

// Long-running thread that manages the HostPool on a timer
func (h *HostPool) manageHostPool(stopper *stoppable.Single) {
	tick := time.Tick(h.poolParams.pruneInterval)
	for {
		select {
		case <-stopper.Quit():
			break
		case <-tick:
			h.ndfMux.RLock()
			if h.isNdfUpdated {
				err := h.updateConns()
				if err != nil {
					jww.ERROR.Printf("Unable to updateConns: %+v", err)
				}
				h.isNdfUpdated = false
			}

			h.hostMux.Lock()
			err := h.pruneHostPool()
			h.hostMux.Unlock()
			if err != nil {
				jww.ERROR.Printf("Unable to pruneHostPool: %+v", err)
			}
			h.ndfMux.RUnlock()
		}
	}
}

// Iterate over the hostList, replacing any empty Hosts or Hosts with errors
// with new, randomly-selected Hosts from the NDF
func (h *HostPool) pruneHostPool() error {
	// Verify the NDF has at least as many Gateways as needed for the HostPool
	ndfLen := uint32(len(h.ndf.Gateways))
	if ndfLen == 0 || ndfLen < h.poolParams.poolSize {
		return errors.Errorf("no gateways available")
	}

	for poolIdx := uint32(0); poolIdx < h.poolParams.poolSize; {
		host := h.hostList[poolIdx]
		// Check the Host for errors
		if host == nil || host.GetMetrics().GetErrorCounter() >= h.poolParams.errThreshold {

			// If errors occurred, randomly select a new Gw by index in the NDF
			ndfIdx := readRangeUint32(0, uint32(len(h.ndf.Gateways)), h.rng)

			// Use the random ndfIdx to obtain a GwId from the NDF
			gwId, err := id.Unmarshal(h.ndf.Gateways[ndfIdx].ID)
			if err != nil {
				return errors.WithMessage(err, "failed to get Gateway for pruning")
			}

			// Verify the GwId is not already in the hostMap
			if _, ok := h.hostMap[*gwId]; !ok {

				// If it is a new GwId, replace the old Host with the new Host
				err = h.replaceHost(gwId, poolIdx)
				if err != nil {
					return err
				}
				poolIdx++
			}
		}
	}
	return nil
}

// Replace the given slot in the HostPool with a new Gateway with the specified ID
func (h *HostPool) replaceHost(newId *id.ID, oldPoolIndex uint32) error {
	// Obtain that GwId's Host object
	newHost, ok := h.manager.GetHost(newId)
	if !ok {
		return errors.Errorf("host for gateway %s could not be "+
			"retrieved", newId)
	}

	// Keep track of oldHost for cleanup
	oldHost := h.hostList[oldPoolIndex]

	// Use the poolIdx to overwrite the random Host in the corresponding index in the hostList
	h.hostList[oldPoolIndex] = newHost
	// Use the GwId to keep track of the new random Host's index in the hostList
	h.hostMap[*newId] = oldPoolIndex

	// Clean up and move onto next Host
	if oldHost != nil {
		delete(h.hostMap, *oldHost.GetId())
		oldHost.Disconnect()
	}
	jww.DEBUG.Printf("Replaced Host at %d with new Host %s", oldPoolIndex, newId.String())
	return nil
}

// Force-add the Gateways to the HostPool, each replacing a random Gateway
func (h *HostPool) ForceAdd(gwIds []*id.ID) error {
	h.hostMux.Lock()
	defer h.hostMux.Unlock()

	checked := make(map[uint32]interface{}) // Keep track of Hosts already replaced
	for i := 0; i < len(gwIds); {
		// Verify the GwId is not already in the hostMap
		if _, ok := h.hostMap[*gwIds[i]]; ok {
			continue
		}

		// Randomly select another Gateway in the HostPool for replacement
		poolIdx := readRangeUint32(0, h.poolParams.poolSize, h.rng)
		if _, ok := checked[poolIdx]; !ok {
			err := h.replaceHost(gwIds[i], poolIdx)
			if err != nil {
				return err
			}
			checked[poolIdx] = nil
			i++
		}
	}
	return nil
}

// Updates the internal HostPool with any changes to the NDF
func (h *HostPool) updateConns() error {
	// Prepare NDFs for comparison
	newMap, err := convertNdfToMap(h.ndf)
	if err != nil {
		return errors.Errorf("Unable to convert new NDF to set: %+v", err)
	}

	// Handle removing Gateways
	for gwId := range h.ndfMap {
		if _, ok := newMap[gwId]; !ok {
			// If GwId in ndfMap is not in newMap, remove the Gateway
			h.removeGateway(&gwId)
		}
	}

	// Handle adding Gateways
	for gwId, ndfIdx := range newMap {
		if _, ok := h.ndfMap[gwId]; !ok {
			// If GwId in newMap is not in ndfMap, add the Gateway
			h.addGateway(&gwId, ndfIdx)
		}
	}

	// Update the internal NDF set
	h.ndfMap = newMap
	return nil
}

// Takes ndf.Gateways and puts their IDs into a map object
func convertNdfToMap(ndf *ndf.NetworkDefinition) (map[id.ID]int, error) {
	result := make(map[id.ID]int)
	// Process gateway Id's into set
	for i, gw := range ndf.Gateways {
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			return nil, err
		}
		result[*gwId] = i
	}

	return result, nil
}

// updateConns helper for removing old Gateways
func (h *HostPool) removeGateway(gwId *id.ID) {
	h.manager.RemoveHost(gwId)
	// If needed, flag the Host for deletion from the HostPool
	if poolIndex, ok := h.hostMap[*gwId]; ok {
		h.hostList[poolIndex] = nil
		delete(h.hostMap, *gwId)
	}
}

// updateConns helper for adding new Gateways
func (h *HostPool) addGateway(gwId *id.ID, ndfIndex int) {
	gw := h.ndf.Gateways[ndfIndex]

	//check if the host exists
	host, ok := h.manager.GetHost(gwId)
	if !ok {

		// Check if gateway ID collides with an existing hard coded ID
		if id.CollidesWithHardCodedID(gwId) {
			jww.ERROR.Printf("Gateway ID invalid, collides with a "+
				"hard coded ID. Invalid ID: %v", gwId.Marshal())
		}

		// Add the new gateway host
		_, err := h.manager.AddHost(gwId, gw.Address, []byte(gw.TlsCertificate), h.poolParams.hostParams)
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

// Get the Host of a random gateway in the NDF
func Get(ndf *ndf.NetworkDefinition, hg HostManager, rng io.Reader) (*connect.Host, error) {
	gwLen := uint32(len(ndf.Gateways))
	if gwLen == 0 {
		return nil, errors.Errorf("no gateways available")
	}

	gwIdx := readRangeUint32(0, gwLen, rng)
	gwID, err := id.Unmarshal(ndf.Nodes[gwIdx].ID)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get Gateway")
	}
	gwID.SetType(id.Gateway)
	gwHost, ok := hg.GetHost(gwID)
	if !ok {
		return nil, errors.Errorf("host for gateway %s could not be "+
			"retrieved", gwID)
	}
	return gwHost, nil
}

// GetAllShuffled returns a shuffled list of gateway hosts from the specified round
func GetAllShuffled(hg HostManager, ri *mixmessages.RoundInfo) ([]*connect.Host, error) {
	roundTop := ri.GetTopology()
	hosts := make([]*connect.Host, 0)
	shuffledList := make([]uint64, 0)

	// Collect all host information from the round
	for index := range roundTop {
		selectedId, err := id.Unmarshal(roundTop[index])
		if err != nil {
			return nil, err
		}

		selectedId.SetType(id.Gateway)

		gwHost, ok := hg.GetHost(selectedId)
		if !ok {
			return nil, errors.Errorf("Could not find host for gateway %s", selectedId)
		}
		hosts = append(hosts, gwHost)
		shuffledList = append(shuffledList, uint64(index))
	}

	returnHosts := make([]*connect.Host, len(hosts))

	// Shuffle a list corresponding to the valid gateway hosts
	shuffle.Shuffle(&shuffledList)

	// Index through the shuffled list, building a list
	// of shuffled gateways from the round
	for index, shuffledIndex := range shuffledList {
		returnHosts[index] = hosts[shuffledIndex]
	}

	return returnHosts, nil

}

// readUint32 reads an integer from an io.Reader (which should be a CSPRNG)
func readUint32(rng io.Reader) uint32 {
	var rndBytes [4]byte
	i, err := rng.Read(rndBytes[:])
	if i != 4 || err != nil {
		panic(fmt.Sprintf("cannot read from rng: %+v", err))
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
