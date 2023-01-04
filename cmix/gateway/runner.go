////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"time"
)

func (hp *hostPool) runner(stop *stoppable.Single) {

	inProgress := make(map[id.ID]struct{})
	toRemoveList := make(map[id.ID]interface{}, 2*cap(hp.writePool.hostList))
	online := newBucket(cap(hp.writePool.hostList))

	for {
		update := false
	input:
		select {
		case <-stop.Quit():
			stop.ToStopped()
			return
		// receives a request to Add a node to the host pool
		// if a specific node if is sent, it will send that id off
		// to testing otherwise, it sends a random one
		case toAdd := <-hp.addRequest:

			var hostList []*connect.Host
			hostList, inProgress = hp.processAddRequest(toAdd, inProgress)
			if len(hostList) == 0 {
				jww.ERROR.Printf("Host list for testing is empty, this " +
					"error should never occur")
				break input
			}

			//send the signal to the adding pool to Add
			select {
			case hp.testNodes <- hostList:
			default:
				jww.ERROR.Printf("Failed to send Add message")
			}
		// handle request to remove a node from the host pool
		case toRemove := <-hp.removeRequest:

			// if the host is already slated to be removed, ignore
			if _, exists := toRemoveList[*toRemove]; exists {
				break input
			}

			// do not remove if it is not present in the pool
			if !hp.writePool.Has(toRemove) {
				jww.DEBUG.Printf("Skipping remove request for %s,"+
					" not in the host pool", toRemove)
				break input
			}

			// add to the leaky bucket detecting if we are offline
			online.Add()

			// Add to the "to remove" list.  This will replace that
			// node on th next addition to the pool
			toRemoveList[*toRemove] = struct{}{}

			//send a signal back to this thread to Add a node to the pool
			go func() {
				hp.addRequest <- nil
			}()

		// internal signal on reception of vetted node to Add to pool
		case newHost := <-hp.newHost:
			// verify the new host is still in the NDF,
			// due to how testing is async, it can get removed
			if _, exists := hp.ndfMap[*newHost.GetId()]; !exists {
				jww.WARN.Printf("New vetted host (%s) is not in NDF, "+
					"this is theoretically possible but extremely unlikely. "+
					"If this is seen more than once, it is likely something is "+
					"wrong", newHost.GetId())
				//send a signal back to this thread to Add a node to the pool
				go func() {
					hp.addRequest <- nil
				}()
				break input
			}

			//
			online.Reset()

			// replace a node slated for replacement if required
			// pop to remove list
			toRemove := pop(toRemoveList)
			if toRemove != nil {
				//if this fails, handle the new host without removing a node
				if oldHost, err := hp.writePool.replaceSpecific(toRemove, newHost); err == nil {
					update = true
					if oldHost != nil {
						go func() {
							oldHost.Disconnect()
						}()
					}
				} else {
					// we can do
					jww.WARN.Printf("Failed to replace %s due to %s, skipping "+
						"addition to host pool", toRemove, err)
				}
			} else {
				stream := hp.rng.GetStream()
				hp.writePool.addOrReplace(stream, newHost)
				stream.Close()

				update = true
			}
		// tested gateways get passed back, so they can be
		// removed from the list of gateways which are being
		// tested
		case tested := <-hp.doneTesting:
			for _, h := range tested {
				delete(inProgress, *h.GetId())
				jww.DEBUG.Printf("[Runner] Deleted %s from inProgress", h.GetId())
			}
		// new NDF updates come in over this channel
		case newNDF := <-hp.newNdf:
			hp.ndf = newNDF.DeepCopy()

			// process the new NDF map
			newNDFMap := hp.processNdf(hp.ndf)

			// remove all gateways which are not missing from the host pool
			// that are in the host pool
			for gwID := range hp.ndfMap {
				if hp.writePool.Has(&gwID) {
					hp.removeRequest <- gwID.DeepCopy()
				}
			}

			// replace the ndfMap
			hp.ndfMap = newNDFMap

		}

		if update == true {
			poolCopy := hp.writePool.deepCopy()
			hp.readPool.Store(poolCopy)

			saveList := make([]*id.ID, 0, len(poolCopy.hostList))
			for i := 0; i < len(poolCopy.hostList); i++ {
				saveList = append(saveList, poolCopy.hostList[i].GetId())
			}

			err := saveHostList(hp.kv, saveList)
			if err != nil {
				jww.WARN.Printf("Host list could not be stored, updates will "+
					"not be available on load: %s", err)
			}
		}

		//Wait the delay until next iteration. this will ensure
		delay := online.GetDelay()
		select {
		case <-time.After(delay):
		case <-stop.Quit():
			stop.ToStopped()
			return
		}

	}

}

func (hp *hostPool) processAddRequest(toAdd *id.ID,
	inProgress map[id.ID]struct{}) ([]*connect.Host, map[id.ID]struct{}) {
	// Get the nodes to Add
	var toTest []*id.ID

	// Add the given ID if it is in the NDF
	if toAdd != nil {
		// Check if it is in the NDF
		if _, exist := hp.ndfMap[*toAdd]; exist {
			toTest = []*id.ID{toAdd}
		}
	}

	// If there are no nodes to Add, randomly select some
	if len(toTest) == 0 {
		var err error
		//if none sent, select random nodes to Add
		stream := hp.rng.GetStream()
		toTest, inProgress, err = hp.writePool.selectNew(stream, hp.ndfMap, inProgress,
			hp.numNodesToTest)
		stream.Close()
		if err != nil {
			jww.DEBUG.Printf("[ProcessAndRequest] SelectNew returned error: %s", err)
			jww.WARN.Printf("Failed to select any nodes to test for adding, " +
				"skipping Add. This error may be the result of being disconnected " +
				"from the internet or very old network credentials")
			return nil, inProgress
		}
	}

	// Get hosts for the selected nodes
	hostList := make([]*connect.Host, 0, len(toTest))
	for i := 0; i < len(toTest); i++ {
		gwID := toTest[i]
		h, exists := hp.manager.GetHost(gwID)
		if !exists {
			jww.FATAL.Panicf("Gateway is not in host pool, this should" +
				"be impossible")
		}
		hostList = append(hostList, h)
	}
	return hostList, inProgress
}

func (hp *hostPool) processNdf(newNdf *ndf.NetworkDefinition) map[id.ID]int {
	newNDFMap := make(map[id.ID]int, len(hp.ndf.Gateways))

	// Make a list of all gateways
	for i := 0; i < len(newNdf.Gateways); i++ {
		gw := newNdf.Gateways[i]

		// Get the ID and bail if it cannot be retrieved
		gwID, err := gw.GetGatewayId()
		if err != nil {
			jww.WARN.Printf("Skipped gateway %d: %x, "+
				"ID couldn't be unmarshalled, %+v", i,
				newNdf.Gateways[i].ID, err)
			continue
		}

		// Skip adding if the node is not active
		if newNdf.Nodes[i].Status != ndf.Active {
			continue
		}

		// Check if the ID exists, if it does not Add its host
		if _, exists := hp.manager.GetHost(gwID); !exists {
			var gwAddr string
			var cert []byte
			gwAddr, cert, err = getConnectionInfo(gwID, gw.Address, gw.TlsCertificate)
			if err != nil {
				_, err = hp.manager.AddHost(gwID, gwAddr,
					cert, hp.params.HostParams)
			}

			if err != nil {
				jww.WARN.Printf("Skipped gateway %d: %s, "+
					"host could not be added, %+v", i,
					gwID, err)
				continue
			}

			hp.addChan <- network.NodeGateway{
				Node:    newNdf.Nodes[i],
				Gateway: gw,
			}

		}

		// Add to the new
		newNDFMap[*gwID] = i

		// delete from the old so we can track which gateways are
		// missing
		delete(hp.ndfMap, *gwID)
	}

	return newNDFMap
}

// pop selects an element from the map that tends to be an earlier insert,
// removes it, and returns it
func pop(m map[id.ID]interface{}) *id.ID {
	for tr := range m {
		delete(m, tr)
		return &tr
	}
	return nil
}
