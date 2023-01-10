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
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/stoppable"
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
	registerRequests := make([]network.NodeGateway, 0, r.bufferSize)

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
				jww.INFO.Printf("timer elapsed")
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
			} else if len(registerRequests) == 1 {
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
			if gateway.IsHostPoolNotReadyError(err) {
				select {
				case <-time.NewTimer(10 * time.Second).C:
				case <-stop.Quit():
					stop.ToStopped()
					return
				}
			}
		}
		registerRequests = make([]network.NodeGateway, 0, r.bufferSize)
	}
}

// registerWithNodes is a helper function which builds a target list and calls
// out to get keys from nodes.  This currently handles the registration
// for precanned identities as an edge case.  Received responses are sent to
// a separate thread for processing
func registerWithNodes(ngws []network.NodeGateway, s session, r *registrar, stop *stoppable.Single) error {
	var toRegister []network.NodeGateway

	// Check each node in the list, handle precanned registration immediately & filter out any registered
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

	if len(toRegister) == 0 {
		return nil
	}

	// Generate diffie-hellman keypair for this batch of registrations
	grp := r.session.GetCmixGroup()
	prime := grp.GetPBytes()
	rng := r.rng.GetStream()
	dhPrivBytes, err := csprng.GenerateInGroup(prime, 32, rng)
	rng.Close()
	if err != nil {
		return err
	}
	dhPriv := grp.NewIntFromBytes(dhPrivBytes)
	dhPub := diffieHellman.GeneratePublicKey(dhPriv, grp)

	// Construct batch request & send to gateway for processing
	signedKeyResponses, err := requestKeys(toRegister, dhPub, s, r, stop)
	if err != nil {
		return err
	}
	// Send responses to channel for processing
	r.rc <- registrationResponse{
		ngws:      ngws,
		responses: signedKeyResponses,
		dhPriv:    dhPriv,
	}

	return nil
}

// Internal type used to pass data from the registration request handler
// to the registration response handler
type registrationResponse struct {
	ngws      []network.NodeGateway
	responses *pb.SignedBatchKeyResponse
	dhPriv    *cyclic.Int
}

// Internal type used to pass data from the registration response handler
// to its worker threads
type registrationResponsePart struct {
	dhPriv   *cyclic.Int
	ngw      network.NodeGateway
	response *pb.SignedKeyResponse
	resChan  chan registrationResponseStatus
}

// Internal type used to pass data from registration response workers to
// the results thread
type registrationResponseStatus struct {
	ngw        network.NodeGateway
	registered bool
	err        error
}

// processNodeRegistrationResponses is a long-running thread which handles
// batch responses received over the rc channel held in registrar.  It starts a
// set number of worker threads, which requests are passed to for processing.
// Results are sent to a separate thread which logs when all have been processed.
func processNodeRegistrationResponses(r *registrar, inProgress, attempts *sync.Map, stop *stoppable.Single) {
	numWorkers := 4
	workerChan := make(chan registrationResponsePart, numWorkers)
	stopWorkers := stoppable.NewMulti("regResponseWorkers")
	// Start worker threads
	for i := 0; i < numWorkers; i++ {
		workerName := fmt.Sprintf("regResponseWorker%d", i)
		stopWorker := stoppable.NewSingle(workerName)
		stopWorkers.Add(stopWorker)
		go func() {
			for {
				select {
				case <-stopWorker.Quit():
					jww.TRACE.Printf("Worker %s received stop signal", workerName)
					return
				case part := <-workerChan:
					registered, err := processResponsePart(part.response, part.ngw, part.dhPriv, r, inProgress, attempts)
					part.resChan <- registrationResponseStatus{
						ngw:        part.ngw,
						registered: registered,
						err:        err,
					}
				}
			}
		}()
	}

	// Main thread reads from response channel, feeds parts to workers
	for {
		select {
		case <-stop.Quit():
			err := stopWorkers.Close()
			if err != nil {
				jww.ERROR.Printf("Failed to close worker threads for registration response processing: %+v", err)
			}
			return
		case resp := <-r.rc:
			resultChan := make(chan registrationResponseStatus, len(resp.ngws))
			for i, ngw := range resp.ngws {
				workerChan <- registrationResponsePart{
					dhPriv:   resp.dhPriv,
					ngw:      ngw,
					response: resp.responses.SignedKeys[i],
					resChan:  resultChan,
				}
				jww.INFO.Printf("Sent %d", i)
			}
			// Start thread which will wait for responses & print results
			go func(l int, c chan registrationResponseStatus) {
				registrationReport := "Batch registration completed - registered with %d/%d nodes"
				detail := "Batch registration details:\n"
				successCount := 0
				for i := 0; i < l; i++ {
					select {
					case res := <-c:
						if res.registered {
							successCount = successCount + 1
						}
						nid, err := res.ngw.Node.GetNodeId()
						if err != nil {
							jww.ERROR.Printf("Failed to unmarshal ID of registration response part: %+v", err)
							continue
						}
						regStatus := ""
						if res.registered {
							regStatus = "registered"
						} else {
							if res.err != nil {
								regStatus = "error"
							} else {
								regStatus = "not registered"
							}
						}
						errDetail := ""
						if res.err != nil {
							errDetail = res.err.Error()
						}
						detail = detail + fmt.Sprintf("\t%s - %s - %s\n", nid.String(), regStatus, errDetail)
					}
				}
				jww.INFO.Printf(registrationReport, successCount, l)
				jww.DEBUG.Print(detail)
			}(len(resp.ngws), resultChan)
		}
	}
}

// processResponsePart is a helper function which accepts and handles a
// SignedKeyResponse. It updates the attempts and inProgress maps &
// adds the received data to the registrar, or returns the node to the
// registration queue as needed
func processResponsePart(resp *pb.SignedKeyResponse, ngw network.NodeGateway, dhPriv *cyclic.Int, r *registrar, inProgress, attempts *sync.Map) (bool, error) {
	nidStr := hex.EncodeToString(ngw.Node.ID)
	nid, err := ngw.Node.GetNodeId()
	if err != nil {
		jww.WARN.Printf(
			"Could not process node ID for registration: %s", err)
		return false, err
	}

	var retry = func(numAttempts uint) {
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
	numAttempts := uint(1)
	if resp == nil {
		numAttemptsInterface, ok := attempts.Load(nidStr)
		if ok {
			numAttempts = numAttemptsInterface.(uint)
		}
		retry(numAttempts)
		return false, nil
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
		retry(numAttempts)
		return false, errors.Errorf("Failed to register node: %s", resp.GetError())
	}

	// Process the result
	transmissionKey, keyId, validUntil, err := processRequestResponse(resp, ngw, r.session.GetCmixGroup(), dhPriv)
	if err != nil {
		retry(numAttempts)
		return false, errors.Errorf("Failed to process batch response part from %s: %+v", nidStr, err)
	}
	r.add(nid, transmissionKey, validUntil, keyId)
	jww.INFO.Printf("Completed registration with node %s", nid)
	return true, nil
}
