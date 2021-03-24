///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"encoding/binary"
	"fmt"
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
	"math"
	"time"
)

type HostGetter interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error)
	RemoveHost(hid *id.ID)
}

type HostPool struct {
	hostMap  map[*id.ID]uint32 // map key to its index in the slice
	hostList []*connect.Host   // each index in the slice contains the value
	ndfSet   *set.Set

	poolSize       uint32
	rng            io.Reader
	ndf            *ndf.NetworkDefinition
	storage        *storage.Session
	getter         HostGetter
	addGatewayChan chan network.NodeGateway
}

//
func NewHostPool(poolSize uint32, rng io.Reader, ndf *ndf.NetworkDefinition, getter HostGetter,
	storage *storage.Session, addGateway chan network.NodeGateway) (*HostPool, error) {
	result := &HostPool{
		getter:         getter,
		hostMap:        make(map[*id.ID]uint32),
		hostList:       make([]*connect.Host, poolSize),
		poolSize:       poolSize,
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

	// Convert the NDF into a set object
	result.ndfSet, err = convertNdfToSet(ndf)
	if err != nil {
		return nil, err
	}

	// Start the long-running thread and return
	go result.manageHostPool()
	return result, nil
}

// Return a random GwHost from the HostPool
func (h *HostPool) GetAny() *connect.Host {
	gwIdx := readRangeUint32(0, h.poolSize, h.rng)
	return h.hostList[gwIdx]
}

// Try to obtain a specific host from the HostPool.
// If that GwId is not in the HostPool, return a random GwHost from the HostPool
func (h *HostPool) GetSpecific(gwId *id.ID) *connect.Host {
	if hostIdx, ok := h.hostMap[gwId]; ok {
		return h.hostList[hostIdx]
	}
	return h.GetAny()
}

// Long-running thread that manages the HostPool on a timer
func (h *HostPool) manageHostPool() {
	// TODO: Configurable timer?
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			err := h.updateConns()
			if err != nil {
				jww.ERROR.Printf("Unable to updateConns: %+v", err)
			}
			err = h.pruneHostPool()
			if err != nil {
				jww.ERROR.Printf("Unable to pruneHostPool: %+v", err)
			}
		}
	}
}

// Iterate over the hostList, replacing any empty Hosts or Hosts with errors
// with new, randomly-selected Hosts from the NDF
func (h *HostPool) pruneHostPool() error {
	// TODO: Verify this logic chunk
	ndfLen := uint32(len(h.ndf.Gateways))
	if ndfLen == 0 || ndfLen < h.poolSize {
		return errors.Errorf("no gateways available")
	}

	for poolIdx := uint32(0); poolIdx < h.poolSize; {
		host := h.hostList[poolIdx]

		// Check the Host for errors
		// TODO: Configurable error threshold?
		if host == nil || *host.GetMetrics().ErrCounter > 0 {
			// If errors occurred, randomly select a new Gw by index in the NDF
			ndfIdx := readRangeUint32(0, uint32(len(h.ndf.Gateways)), h.rng)
			// Use the random ndfIdx to obtain a GwId from the NDF
			gwId, err := id.Unmarshal(h.ndf.Gateways[ndfIdx].ID)
			if err != nil {
				return errors.WithMessage(err, "failed to get Gateway for pruning")
			}

			// Verify the GwId is not already in the hostMap
			if _, ok := h.hostMap[gwId]; !ok {
				// If it is a new GwId, then obtain that GwId's Host object
				gwHost, ok := h.getter.GetHost(gwId)
				if !ok {
					return errors.Errorf("host for gateway %s could not be "+
						"retrieved", gwId)
				}

				// Use the poolIdx to overwrite the random Host in the corresponding index in the hostList
				h.hostList[poolIdx] = gwHost
				// Use the GwId to keep track of the new random Host's index in the hostList
				h.hostMap[gwId] = poolIdx

				// Clean up and move onto next Host
				if host != nil {
					host.Disconnect()
				}
				poolIdx++
			}
		}
	}
	return nil
}

// Updates the internal HostPool with any changes to the NDF
func (h *HostPool) updateConns() error {
	// Prepare NDFs for comparison
	newSet, err := convertNdfToSet(h.ndf)
	if err != nil {
		return errors.Errorf("Unable to convert new NDF to set: %+v", err)
	}
	unchangedSet := h.ndfSet.Intersection(newSet)

	// Handle removing Gateways
	removeSet := h.ndfSet.Difference(unchangedSet)
	removeSet.Do(h.removeGateway)

	// Handle adding Gateways
	addSet := newSet.Difference(unchangedSet)
	addSet.Do(h.addGateway)

	// Update the internal NDF set
	h.ndfSet = newSet
	return nil
}

// Used to store information when converting ndf object to set object
type SetItem struct {
	GatewayId *id.ID
	NdfIndex  uint32
}

// Takes ndf.Gateways and puts their IDs into a set.Set object
func convertNdfToSet(ndf *ndf.NetworkDefinition) (*set.Set, error) {
	ndfSet := set.New()

	// Process gateway Id's into set
	for i, gw := range ndf.Gateways {
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			return nil, err
		}
		ndfSet.Insert(SetItem{
			GatewayId: gwId,
			NdfIndex:  uint32(i),
		})
	}

	return ndfSet, nil
}

// updateConns helper for removing old Gateways
func (h *HostPool) removeGateway(setItem interface{}) {
	gwItem := setItem.(SetItem)
	h.getter.RemoveHost(gwItem.GatewayId)
}

// updateConns helper for adding new Gateways
func (h *HostPool) addGateway(setItem interface{}) {
	gwItem := setItem.(SetItem)
	gw := h.ndf.Gateways[gwItem.NdfIndex]

	//check if the host exists
	host, ok := h.getter.GetHost(gwItem.GatewayId)
	if !ok {

		// Check if gateway ID collides with an existing hard coded ID
		if id.CollidesWithHardCodedID(gwItem.GatewayId) {
			jww.ERROR.Printf("Gateway ID invalid, collides with a "+
				"hard coded ID. Invalid ID: %v", gwItem.GatewayId.Marshal())
		}

		// TODO: Make configurable
		gwParams := connect.GetDefaultHostParams()
		gwParams.MaxRetries = 3
		gwParams.EnableCoolOff = true
		_, err := h.getter.AddHost(gwItem.GatewayId, gw.Address, []byte(gw.TlsCertificate), gwParams)
		if err != nil {
			jww.ERROR.Printf("Could not add gateway host %s: %+v", gwItem.GatewayId, err)
		}

		// Send AddGateway event if we do not already possess keys for the GW
		if !h.storage.Cmix().Has(gwItem.GatewayId) {
			ng := network.NodeGateway{
				Node:    h.ndf.Nodes[gwItem.NdfIndex],
				Gateway: gw,
			}

			select {
			case h.addGatewayChan <- ng:
			default:
				jww.WARN.Printf("Unable to send AddGateway event for id %s", gwItem.GatewayId.String())
			}
		}

	} else if host.GetAddress() != gw.Address {
		host.UpdateAddress(gw.Address)
	}
}

// Get the Host of a random gateway in the NDF
func Get(ndf *ndf.NetworkDefinition, hg HostGetter, rng io.Reader) (*connect.Host, error) {
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
func GetAllShuffled(hg HostGetter, ri *mixmessages.RoundInfo) ([]*connect.Host, error) {
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
