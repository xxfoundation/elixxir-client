////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"crypto/sha256"
	"encoding/hex"
	"gitlab.com/xx_network/crypto/csprng"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/ndf"
)

// registerNodes is a manager thread which waits on a channel for nodes
// to register with. On reception, it tries to register with that node.
// This thread is interrupted by the stoppable.Single passed in.
// The sync.Map's keep track of the node(s) that were in progress
// before an interruption and how many registration attempts have
// been attempted.
func registerNodes(r *registrar, s session, stop *stoppable.Single,
	inProgress, attempts *sync.Map) {

	atomic.AddInt64(r.numberRunning, 1)

	for {
		select {
		case <-r.pauser:
			atomic.AddInt64(r.numberRunning, -1)
			select {
			case <-stop.Quit():
				stop.ToStopped()
				return
			case <-r.resumer:
				atomic.AddInt64(r.numberRunning, 1)
			}
		case <-stop.Quit():
			stop.ToStopped()
			atomic.AddInt64(r.numberRunning, -1)
			return

		case gw := <-r.c:
			rng := r.rng.GetStream()

			// Pull node information from channel
			nidStr := hex.EncodeToString(gw.Node.ID)
			nid, err := gw.Node.GetNodeId()
			if err != nil {
				jww.WARN.Printf(
					"Could not process node ID for registration: %s", err)
				rng.Close()
				continue
			}

			// Check if the registrar has this node already
			if r.HasNode(nid) {
				jww.TRACE.Printf(
					"Not registering node %s, already registered", nid)
			}

			// Check if the client is already attempting to register with this
			// node in another thread
			if _, operating := inProgress.LoadOrStore(nidStr,
				struct{}{}); operating {
				rng.Close()
				continue
			}

			// Keep track of how many times registering with this node
			// has been attempted
			numAttempts := uint(1)
			if nunAttemptsInterface, hasValue := attempts.LoadOrStore(
				nidStr, numAttempts); hasValue {
				numAttempts = nunAttemptsInterface.(uint)
				attempts.Store(nidStr, numAttempts+1)
			}

			// No need to register with stale nodes
			if isStale := gw.Node.Status == ndf.Stale; isStale {
				jww.DEBUG.Printf(
					"Skipping registration with stale nodes %s", nidStr)
				rng.Close()
				continue
			}

			// Register with this node
			err = registerWithNode(r.sender, r.comms, gw, s, r, rng, stop)

			// Remove from in progress immediately (success or failure)
			inProgress.Delete(nidStr)

			// Process the result
			if err != nil {
				jww.ERROR.Printf("Failed to register node: %s", err.Error())
				// If we have not reached the attempt limit for this gateway,
				// then send it back into the channel to retry
				if numAttempts < maxAttempts {
					go func() {
						// Delay the send operation for a backoff
						time.Sleep(delayTable[numAttempts-1])
						r.c <- gw
					}()
				}
			}
			rng.Close()
		}
	}
}

// registerWithNode serves as a helper for registerNodes. It registers a user
// with a specific in the client's NDF.
func registerWithNode(sender gateway.Sender, comms RegisterNodeCommsInterface,
	ngw network.NodeGateway, s session, r *registrar,
	rng csprng.Source, stop *stoppable.Single) error {

	nodeID, err := ngw.Node.GetNodeId()
	if err != nil {
		jww.ERROR.Printf("registerWithNode failed to decode node ID: %v", err)
		return err
	}

	if r.HasNode(nodeID) {
		return nil
	}

	jww.INFO.Printf("registerWithNode begin registration with node: %s",
		nodeID)

	var transmissionKey *cyclic.Int
	var validUntil uint64
	var keyId []byte

	start := time.Now()
	// TODO: should move this to a pre-canned user initialization
	if s.IsPrecanned() {
		userNum := int(s.GetTransmissionID().Bytes()[7])
		h := sha256.New()
		h.Reset()
		h.Write([]byte(strconv.Itoa(4000 + userNum)))

		transmissionKey = r.session.GetCmixGroup().NewIntFromBytes(h.Sum(nil))
		jww.INFO.Printf("transmissionKey: %v", transmissionKey.Bytes())
	} else {
		// Request key from server
		transmissionKey, keyId, validUntil, err = requestKey(
			sender, comms, ngw, s, r, rng, stop)

		if err != nil {
			return errors.Errorf("Failed to request key: %v", err)
		}

	}

	r.add(nodeID, transmissionKey, validUntil, keyId)

	jww.INFO.Printf("Completed registration with node %s,"+
		" took %d", nodeID, time.Since(start))

	return nil
}
