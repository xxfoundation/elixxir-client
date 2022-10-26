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
	"gitlab.com/xx_network/crypto/csprng"
	"strconv"
	"sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
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
func processNodeRegistration(r *registrar, s session, stop *stoppable.Single,
	inProgress, attempts *sync.Map) {
	timerCh := make(<-chan time.Time)
	var registerRequests []network.NodeGateway

	for {
		shouldProcess := false

		select {
		case <-stop.Quit():
			for _, req := range registerRequests {
				select {
				case r.c <- req:
				default:
				}
			}

			// On a stop signal, close the thread
			stop.ToStopped()
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
				continue
			}
			registerRequests = append(registerRequests, gw)

			if len(registerRequests) > int(25) { // TODO parametrize this
				shouldProcess = true
			} else if len(registerRequests) == 1 { // TODO this was orgiinally != 0 in historical rounds, am i missing something?
				// If this is the first round, start the timeout
				timerCh = time.NewTimer(100 * time.Millisecond).C // TODO parametrize this
			}

		}

		if !shouldProcess {
			continue
		}
		rng := r.rng.GetStream()
		err := registerWithNodes(registerRequests, s, r, rng, stop)
		if err != nil {
			jww.ERROR.Printf("Failed to register with batch of nodes %+v: %+v", registerRequests, err)
		}
		rng.Close()
	}
}

func registerWithNodes(ngws []network.NodeGateway, s session, r *registrar,
	rng csprng.Source, stop *stoppable.Single) error {
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

		jww.INFO.Printf("registerWithNode begin registration with node: %s",
			nodeID)

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
	return requestKeys(r.sender, r.comms, toRegister, s, r, rng, stop)
}

type registrationResponsePart struct {
	ngw      network.NodeGateway
	response *pb.SignedKeyResponse
	dhPriv   *cyclic.Int
}

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

			if resp.Error != "" {
				jww.ERROR.Printf("Received error in batch registration response for gw %s: %s", nidStr, resp.Error)
				continue
			}

			// Process the result
			transmissionKey, keyId, validUntil, err := processRequestResponse(resp, respPart.ngw, grp, respPart.dhPriv)
			if err != nil {
				jww.ERROR.Printf("Failed to process batch response part from %s: %+v", nidStr, err)
			}
			r.add(nid, transmissionKey, validUntil, keyId)
			jww.INFO.Printf("Completed registration with node %s", nid)

			// Remove from in progress immediately (success or failure)
			inProgress.Delete(nidStr)

			if err != nil {
				jww.ERROR.Printf("Failed to register node: %s", err.Error())
				// If we have not reached the attempt limit for this gateway,
				// then send it back into the channel to retry
				numAttempts, ok := attempts.Load(nidStr)
				if !ok {
					jww.ERROR.Printf("This shoudl not happen")
				}
				if numAttempts.(uint) < maxAttempts {
					go func() {
						// Delay the send operation for a backoff
						time.Sleep(delayTable[numAttempts.(uint)-1])
						r.c <- ngw
					}()
				}
			}
		}
	}
}
