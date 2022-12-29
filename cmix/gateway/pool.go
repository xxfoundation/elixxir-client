////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"io"
)

var errHostPoolNotReady = errors.New("Host pool is not ready, wait a " +
	"little then try again. if this persists, you may have connectivity issues")

type Pool interface {
	Get(id *id.ID) (*connect.Host, bool)
	Has(id *id.ID) bool
	Size() int
	IsReady() error
	GetAny(length uint32, excluded []*id.ID, rng io.Reader) []*connect.Host
	GetSpecific(target *id.ID) (*connect.Host, bool)
	GetPreferred(targets []*id.ID, rng io.Reader) []*connect.Host
}

type pool struct {
	hostMap     map[id.ID]uint  // Map key to its index in the slice
	hostList    []*connect.Host // Each index in the slice contains the value
	isConnected func(host *connect.Host) bool
}

// newPool creates a pool of size "size"
func newPool(size int) *pool {
	return &pool{
		hostMap:  make(map[id.ID]uint, size),
		hostList: make([]*connect.Host, 0, size),
		isConnected: func(host *connect.Host) bool {
			c, _ := host.Connected()
			return c
		},
	}
}

// Get returns a specific member of the host pool if it exists
func (p *pool) Get(id *id.ID) (*connect.Host, bool) {
	h, exists := p.hostMap[*id]
	return p.hostList[h], exists
}

// Has returns true if the id is a member of the host pool
func (p *pool) Has(id *id.ID) bool {
	_, exists := p.hostMap[*id]
	return exists
}

// Size returns the number of currently connected and usable members
// of the host pool
func (p *pool) Size() int {
	size := 0
	for i := 0; i < len(p.hostList); i++ {
		if p.isConnected(p.hostList[i]) {
			size++
		}
	}
	return size
}

// IsReady returns true if there is at least one connected member of the hostPool
func (p *pool) IsReady() error {
	jww.TRACE.Printf("[IsReady] Length of Host List %d", len(p.hostList))
	for i := 0; i < len(p.hostList); i++ {
		if p.isConnected(p.hostList[i]) {
			return nil
		}
	}
	return errHostPoolNotReady
}

// GetAny returns up to n host pool members randomly, excluding any which are
// in the host pool list. if the number of returnable members is less than
// the number requested, is will return the smaller set
// will not return any disconnected hosts
func (p *pool) GetAny(length uint32, excluded []*id.ID, rng io.Reader) []*connect.Host {

	poolLen := uint32(len(p.hostList))
	if length > poolLen {
		length = poolLen
	}

	// Keep track of Hosts already selected to avoid duplicates
	checked := make(map[uint32]interface{}, len(p.hostList))
	if excluded != nil {
		// Add excluded Hosts to already-checked list
		for i := range excluded {
			gwId := excluded[i]
			if idx, ok := p.hostMap[*gwId]; ok {
				checked[uint32(idx)] = nil
			}
		}
	}

	result := make([]*connect.Host, 0, length)
	for i := uint32(0); i < length; {
		// If we've checked the entire HostPool, bail
		if uint32(len(checked)) >= poolLen {
			break
		}

		// Check the next HostPool index
		gwIdx := randomness.ReadRangeUint32(0, poolLen,
			rng)

		if _, ok := checked[gwIdx]; !ok {
			h := p.hostList[gwIdx]

			checked[gwIdx] = nil
			if !p.isConnected(h) {
				continue
			}

			result = append(result, p.hostList[gwIdx])
			i++
		}
	}

	return result
}

// GetSpecific obtains a specific connect.Host from the pool if it exists,
// it otherwise returns nil (and false on the bool) if it does not.
// It will not return the host if it is in the pool but disconnected
func (p *pool) GetSpecific(target *id.ID) (*connect.Host, bool) {
	if idx, exists := p.hostMap[*target]; exists {
		h := p.hostList[idx]
		if !p.isConnected(h) {
			return nil, false
		}
		return h, true
	}
	return nil, false
}

// GetPreferred tries to obtain the given targets from the HostPool. If each is
// not present, then obtains a random replacement from the HostPool which will
// be proxied.
func (p *pool) GetPreferred(targets []*id.ID, rng io.Reader) []*connect.Host {
	// Keep track of Hosts already selected to avoid duplicates
	checked := make(map[id.ID]struct{})

	//edge checks
	numToReturn := len(targets)
	poolLen := len(p.hostList)
	if numToReturn > poolLen {
		numToReturn = poolLen
	}
	result := make([]*connect.Host, 0, numToReturn)

	//check if any targets are in the pool
	numSelected := 0
	for _, target := range targets {
		if targeted, inPool := p.GetSpecific(target); inPool {
			result = append(result, targeted)
			checked[*target] = struct{}{}
			numSelected++
		}
	}

	//fill the rest of the list with random proxies until full
	for numSelected < numToReturn && len(checked) < len(p.hostList) {

		gwIdx := randomness.ReadRangeUint32(0, uint32(len(p.hostList)),
			rng)
		selected := p.hostList[gwIdx]
		//check if it is already in the list, if not Add it
		gwID := selected.GetId()
		if _, ok := checked[*gwID]; !ok {
			checked[*gwID] = struct{}{}
			if !p.isConnected(selected) {
				continue
			}
			result = append(result, selected)
			numSelected++
		}
	}

	return result
}

// addOrReplace adds the given host if the pool is not full, or replaces a
// random one if the pool is full. If a host was replaced, it returns it, so
// it can be cleaned up.
func (p *pool) addOrReplace(rng io.Reader, host *connect.Host) *connect.Host {
	// if the pool is not full, append to the end
	if len(p.hostList) < cap(p.hostList) {
		jww.TRACE.Printf("[AddOrReplace] Adding host %s to host list", host.GetId())
		p.hostList = append(p.hostList, host)
		p.hostMap[*host.GetId()] = uint(len(p.hostList) - 1)
		return nil
	} else {
		jww.TRACE.Printf("[AddOrReplace] Internally replacing...")
		selectedIndex := uint(randomness.ReadRangeUint32(0, uint32(len(p.hostList)), rng))
		return p.internalReplace(selectedIndex, host)
	}
}

// replaceSpecific will replace a specific gateway with the given ID with
// the given host.
func (p *pool) replaceSpecific(toReplace *id.ID,
	host *connect.Host) (*connect.Host, error) {
	selectedIndex, exists := p.hostMap[*toReplace]
	if !exists {
		return nil, errors.Errorf("Cannot replace %s, host does not "+
			"exist in pool", toReplace)
	}
	return p.internalReplace(selectedIndex, host), nil
}

// internalReplace places the given host into the hostList and the hostMap.
// This will replace the data from the given index.
func (p *pool) internalReplace(selectedIndex uint, host *connect.Host) *connect.Host {
	toRemove := p.hostList[selectedIndex]
	p.hostList[selectedIndex] = host
	delete(p.hostMap, *toRemove.GetId())
	p.hostMap[*host.GetId()] = selectedIndex
	return toRemove
}

// deepCopy returns a deep copy of the internal state of pool.
func (p *pool) deepCopy() *pool {
	pCopy := &pool{
		hostMap:     make(map[id.ID]uint, len(p.hostMap)),
		hostList:    make([]*connect.Host, len(p.hostList)),
		isConnected: p.isConnected,
	}

	copy(pCopy.hostList, p.hostList)

	for key, data := range p.hostMap {
		pCopy.hostMap[key] = data
	}

	return pCopy
}

// selectNew will pull random nodes from the pool.
func (p *pool) selectNew(rng csprng.Source, allNodes map[id.ID]int,
	currentlyAddingNodes map[id.ID]struct{}, numToSelect int) ([]*id.ID,
	map[id.ID]struct{}, error) {

	newList := make(map[id.ID]interface{})

	// Copy all nodes while removing nodes from the host list and
	// from the processing list
	for nid := range allNodes {
		_, inPool := p.hostMap[nid]
		_, inAdd := currentlyAddingNodes[nid]
		if !(inPool || inAdd) {
			newList[nid] = struct{}{}
		}
	}

	// Error out if no nodes are left
	if len(newList) == 0 {
		return nil, nil, errors.New("no nodes available for selection")
	}

	if numToSelect > len(newList) {
		// Return all nodes
		selections := make([]*id.ID, 0, len(newList))
		for gwID := range newList {
			localGwid := gwID.DeepCopy()
			selections = append(selections, localGwid)
			currentlyAddingNodes[*localGwid] = struct{}{}
			jww.DEBUG.Printf("[SelectNew] Adding gwId %s to inProgress", localGwid)
		}
		return selections, currentlyAddingNodes, nil
	}

	// Randomly select numToSelect indices
	toSelectMap := make(map[uint]struct{}, numToSelect)
	for i := 0; i < numToSelect; i++ {
		newSelection := uint(randomness.ReadRangeUint32(0, uint32(len(newList)), rng))
		if _, exists := toSelectMap[newSelection]; exists {
			i--
			continue
		}
		toSelectMap[newSelection] = struct{}{}
	}

	// Use the random indices to choose gateways
	selections := make([]*id.ID, 0, numToSelect)
	// Select the new ones
	index := uint(0)
	for gwID := range newList {
		localGwid := gwID.DeepCopy()
		if _, exists := toSelectMap[index]; exists {
			selections = append(selections, localGwid)
			currentlyAddingNodes[*localGwid] = struct{}{}
			if len(selections) == cap(selections) {
				break
			}
		}
		index++
	}

	return selections, currentlyAddingNodes, nil
}
