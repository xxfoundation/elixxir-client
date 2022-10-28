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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/ndf"
)

// registerNodes is a manager thread which waits on a channel for nodes
// to register with. On reception, nodes are added to a batch pending
// registration.  When either the batch is full, or enough time has elapsed
// since the first node was added to the batch, the request is sent to a
// gateway for processing. This thread is interrupted by the stoppable.Single
// passed in. The sync.Map's keep track of the node(s) that were in progress
// before an interruption and how many registration attempts have
// been attempted.
func processNodeRegistration(r *registrar, s session, stop *stoppable.Single,
	inProgress, attempts *sync.Map, index int) {
	timerCh := make(<-chan time.Time)
	var registerRequests []network.NodeGateway

	atomic.AddInt64(r.numberRunning, 1)

	for {
		shouldProcess := false

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
			// On a stop signal re-add all requests in batch & close the thread
			for _, req := range registerRequests {
				select {
				case r.c <- req:
				default:
				}
			}

			stop.ToStopped()
			atomic.AddInt64(r.numberRunning, -1)
			return
		case <-timerCh:
			// If timer elapses and any register requests exist, process them
			// This avoids too much delay
			if len(registerRequests) > 0 {
				shouldProcess = true
			}
		case gw := <-r.c:
			// Pull node information from channel
			nidStr := hex.EncodeToString(gw.Node.ID)
			nid, err := gw.Node.GetNodeId()
			if err != nil {
				jww.WARN.Printf(
					"Could not process node ID for registration: %s", err)
				continue
			}

			jww.DEBUG.Printf("Received request to register with %s", nidStr)

			// Check if the registrar has this node already
			if r.HasNode(nid) {
				jww.TRACE.Printf(
					"Not registering node %s, already registered", nid)
			}

			// Check if the client is already attempting to register with this
			// node in another thread
			if _, operating := inProgress.LoadOrStore(nidStr,
				struct{}{}); operating {
				continue
			}

			// No need to register with stale nodes
			if isStale := gw.Node.Status == ndf.Stale; isStale {
				jww.DEBUG.Printf(
					"Skipping registration with stale nodes %s", nidStr)
				continue
			}

			// Add nodegateway to batch for processing
			registerRequests = append(registerRequests, gw)

			if len(registerRequests) >= int(r.bufferSize) {
				// Mark for processing if batch is full
				shouldProcess = true
			} else if len(registerRequests) == 1 { // TODO this was != 0 in historical rounds, am i missing something?
				// If this is the first round, start the timeout
				timerCh = time.NewTimer(time.Duration(r.batchDelay) * time.Millisecond).C
			}

		}

		if !shouldProcess {
			continue
		}

		err := registerWithNodes(registerRequests, s, r, stop)
		if err != nil {
			for _, ngw := range registerRequests {
				nidStr := hex.EncodeToString(ngw.Node.ID)

				inProgress.Delete(nidStr)

				// If we have not reached the attempt limit for this gateway,
				// then send it back into the channel to retry
				numAttempts := uint(1)
				numAttemptsInterface, ok := attempts.Load(nidStr)
				if ok {
					numAttempts = numAttemptsInterface.(uint)
				}
				if numAttempts < maxAttempts {
					toRetry := ngw // In the loop, so we need a scoped variable in the gofunc
					go func() {
						// Delay the send operation for a backoff
						time.Sleep(delayTable[numAttempts-1])
						r.c <- toRetry
					}()
				}
			}
			jww.ERROR.Printf("Failed to register with batch of nodes %+v: %+v", registerRequests, err)
		}
		registerRequests = []network.NodeGateway{}
		if index >= 2 {
			if float64(r.NumRegisteredNodes()) > (float64(r.numnodesGetter()) * .7) {
				<-stop.Quit()
				stop.ToStopped()
				return
			}
		}
	}
}

// registerWithNodes is a helper function which builds a target list and calls
// out to get keys from nodes.  This currently handles the registration
// for precanned identities as an edge case
func registerWithNodes(ngws []network.NodeGateway, s session, r *registrar, stop *stoppable.Single) error {
	var toRegister []network.NodeGateway

	for _, ngw := range ngws {
		// Register with this node
		nodeID, err := ngw.Node.GetNodeId()
		if err != nil {
			jww.ERROR.Printf("registerWithNode failed to decode node ID: %v", err)
			continue
		}

		if r.HasNode(nodeID) {
			continue
		}

		var transmissionKey *cyclic.Int
		var validUntil uint64
		var keyId []byte

		// TODO: should move this to a pre-canned user initialization
		// TODO how should we handle precanned with batch reg?
		if s.IsPrecanned() {
			userNum := int(s.GetTransmissionID().Bytes()[7])
			h := sha256.New()
			h.Reset()
			h.Write([]byte(strconv.Itoa(4000 + userNum)))

			transmissionKey = r.session.GetCmixGroup().NewIntFromBytes(h.Sum(nil))
			jww.INFO.Printf("preCanned transmissionKey: %v", transmissionKey.Bytes())
			r.add(nodeID, transmissionKey, validUntil, keyId)
		} else {
			toRegister = append(toRegister, ngw)
		}
	}
	return requestKeys(toRegister, s, r, stop)
}

type registrationResponsePart struct {
	ngw      network.NodeGateway
	response *pb.SignedKeyResponse
	dhPriv   *cyclic.Int
}

// processNodeRegistrationResponses is a long-running thread which handles
//responses received over the rc channel held in registrar.  As registration
// responses are received, it updates the attempts and inProgress maps &
// adds the received data to the registrar, or returns the node to the
// registration queue as needed
func processNodeRegistrationResponses(r *registrar, inProgress, attempts *sync.Map, stop *stoppable.Single) {
	grp := r.session.GetCmixGroup()
	for {
		select {
		case <-stop.Quit():
			return
		case respPart := <-r.rc:
			resp := respPart.response
			ngw := respPart.ngw

			nidStr := hex.EncodeToString(ngw.Node.ID)
			nid, err := ngw.Node.GetNodeId()
			if err != nil {
				jww.WARN.Printf(
					"Could not process node ID for registration: %s", err)
				continue
			}
			jww.DEBUG.Printf("Processing registration response from %s", nid)

			numAttempts := uint(1)
			var retry = func() {
				// If we have not reached the attempt limit for this gateway,
				// then send it back into the channel to retry
				if numAttempts < maxAttempts {
					go func() {
						// Delay the send operation for a backoff
						time.Sleep(delayTable[numAttempts-1])
						r.c <- ngw
					}()
				}
			}

			// Gateway did not have contact with target
			if resp == nil {
				numAttemptsInterface, ok := attempts.Load(nidStr)
				if ok {
					numAttempts = numAttemptsInterface.(uint)
				}
				retry()
				continue
			}

			// Keep track of how many times registering with this node
			// has been attempted
			if numAttemptsInterface, hasValue := attempts.LoadOrStore(
				nidStr, numAttempts); hasValue {
				numAttempts = numAttemptsInterface.(uint)
				attempts.Store(nidStr, numAttempts+1)
			}

			// Remove from in progress immediately (success or failure)
			inProgress.Delete(nidStr)

			if resp.GetError() != "" {
				jww.ERROR.Printf("Failed to register node: %s", resp.GetError())
				retry()
				continue
			}

			// Process the result
			transmissionKey, keyId, validUntil, err := processRequestResponse(resp, respPart.ngw, grp, respPart.dhPriv)
			if err != nil {
				jww.ERROR.Printf("Failed to process batch response part from %s: %+v", nidStr, err)
				retry()
				continue
			}
			r.add(nid, transmissionKey, validUntil, keyId)
			jww.INFO.Printf("Completed registration with node %s", nid)
		}
	}
}
