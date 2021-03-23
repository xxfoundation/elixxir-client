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
	"gitlab.com/elixxir/comms/mixmessages"
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
}

type HostPool struct {
	hostMap  map[*id.ID]uint32 // map key to its index in the slice
	hostList []*connect.Host   // each index in the slice contains the value
	ndfSet   *set.Set

	poolSize uint32
	rng      io.Reader
	ndf      *ndf.NetworkDefinition
	getter   HostGetter
}

func NewHostPool(poolSize uint32, rng io.Reader, ndf *ndf.NetworkDefinition, getter HostGetter) (*HostPool, error) {
	result := &HostPool{
		getter:   getter,
		hostMap:  make(map[*id.ID]uint32),
		hostList: make([]*connect.Host, poolSize),
		poolSize: poolSize,
		ndf:      ndf,
		rng:      rng,
	}

	// Convert the gate
	var err error
	result.ndfSet, err = convertNdfToSet(ndf)
	if err != nil {
		return nil, err
	}

	// Build the initial HostPool
	err = result.buildHostPool()
	if err != nil {
		return nil, err
	}
	// Start the long-running thread and return
	result.manageHostPool()
	return result, nil
}

// Long-running thread that manages the HostPool on a timer
func (g *HostPool) manageHostPool() {
	// TODO: Configurable timer?
	tick := time.Tick(10 * time.Second)
	for {
		select {
		case <-tick:
			err := g.pruneHostPool()
			jww.ERROR.Printf("Unable to prune host pool: %+v", err)
			// TODO: UpdateGwConns functionality here
		}
	}
}

// Iterate over the hostList, replacing any Hosts with errors
// with new, randomly-selected Hosts from the NDF
func (g *HostPool) pruneHostPool() error {
	for poolIdx := uint32(0); poolIdx < g.poolSize; {
		host := g.hostList[poolIdx]
		errCounter := *host.GetMetrics().ErrCounter

		// Check the Host for errors
		// TODO: Configurable error threshold?
		if errCounter > 0 {
			// If errors occurred, randomly select a new Gw by index in the NDF
			ndfIdx := readRangeUint32(0, uint32(len(g.ndf.Gateways)), g.rng)
			// Use the random ndfIdx to obtain a GwId from the NDF
			gwId, err := id.Unmarshal(g.ndf.Gateways[ndfIdx].ID)
			if err != nil {
				return errors.WithMessage(err, "failed to get Gateway for pruning")
			}

			// Verify the GwId is not already in the hostMap
			if _, ok := g.hostMap[gwId]; !ok {
				// If it is a new GwId, then obtain that GwId's Host object
				gwHost, ok := g.getter.GetHost(gwId)
				if !ok {
					return errors.Errorf("host for gateway %s could not be "+
						"retrieved", gwId)
				}

				// Use the poolIdx to overwrite the random Host in the corresponding index in the hostList
				g.hostList[poolIdx] = gwHost
				// Use the GwId to keep track of the new random Host's index in the hostList
				g.hostMap[gwId] = poolIdx

				// Clean up and move onto next Host
				host.Disconnect()
				poolIdx++
			}
		}
	}
	return nil
}

// Create the initial hostList and hostMap
func (g *HostPool) buildHostPool() error {
	// TODO: Verify this logic chunk
	ndfLen := uint32(len(g.ndf.Gateways))
	if ndfLen == 0 || ndfLen < g.poolSize {
		return errors.Errorf("no gateways available")
	}

	// Map random NDF indexes to indexes in the hostList
	indices := make(map[uint32]uint32) // map[ndfIdx]poolIdx
	for poolIdx := uint32(0); poolIdx < g.poolSize; {

		// Randomly select a Gw by index in the NDF
		ndfIdx := readRangeUint32(0, ndfLen, g.rng)

		// If that ndfIdx has already been chosen, skip and try again
		if _, ok := indices[ndfIdx]; !ok {
			// If not, record the Gw NDF index corresponding to its index in the hostList
			indices[ndfIdx] = poolIdx
			poolIdx++
		}
	}

	for ndfIdx, poolIdx := range indices {

		// Use the random ndfIdx to obtain a GwId from the NDF
		gwId, err := id.Unmarshal(g.ndf.Gateways[ndfIdx].ID)
		if err != nil {
			return errors.WithMessage(err, "failed to get Gateway")
		}

		// Then obtain that GwId's Host object
		gwHost, ok := g.getter.GetHost(gwId)
		if !ok {
			return errors.Errorf("host for gateway %s could not be "+
				"retrieved", gwId)
		}

		// Use the poolIdx to assign the random Host to an index in the hostList
		g.hostList[poolIdx] = gwHost
		// Use the GwId to keep track of the Host's random index in the hostList
		g.hostMap[gwId] = poolIdx
	}
	return nil
}

// Return a random GwHost from the HostPool
func (g *HostPool) GetAny() *connect.Host {
	gwIdx := readRangeUint32(0, g.poolSize, g.rng)
	return g.hostList[gwIdx]
}

// Try to obtain a specific host from the HostPool.
// If that GwId is not in the HostPool, return a random GwHost from the HostPool
func (g *HostPool) GetSpecific(gwId *id.ID) *connect.Host {
	if hostIdx, ok := g.hostMap[gwId]; ok {
		return g.hostList[hostIdx]
	}
	return g.GetAny()
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

// Takes ndf.Gateways and puts their IDs into a set.Set object
func convertNdfToSet(ndf *ndf.NetworkDefinition) (*set.Set, error) {
	ndfSet := set.New()

	// Process gateway Id's into set
	for _, gw := range ndf.Gateways {
		gwId, err := id.Unmarshal(gw.ID)
		if err != nil {
			return nil, err
		}
		ndfSet.Insert(gwId)
	}

	return ndfSet, nil
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
