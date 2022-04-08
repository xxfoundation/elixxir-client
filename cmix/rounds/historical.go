///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// Historical Rounds looks up the round history via random gateways. It batches
// these quests but never waits longer than params.HistoricalRoundsPeriod to d
// a lookup.
// Historical rounds receives input from:
//   - Network Follower (/network/follow.go)
// Historical Rounds sends the output to:
//	 - Message Retrieval Workers (/network/round/retrieve.go)

type Retriever interface {
	StartProcesses() *stoppable.Single
	LookupHistoricalRound(rid id.Round, callback RoundResultCallback) error
}

// manager is the controlling structure.
type manager struct {
	params Params

	comms  Comms
	sender gateway.Sender
	events event.Manager

	c chan roundRequest
}

// Comms interface to increase east of testing of historical rounds.
type Comms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestHistoricalRounds(host *connect.Host, message *pb.HistoricalRounds) (
		*pb.HistoricalRoundsResponse, error)
}

// RoundResultCallback is the used callback when a round is found.
type RoundResultCallback func(round Round, success bool)

// roundRequest is an internal structure that tracks a request.
type roundRequest struct {
	rid id.Round
	RoundResultCallback
	numAttempts uint
}

func NewRetriever(param Params, comms Comms, sender gateway.Sender,
	events event.Manager) Retriever {
	return &manager{
		params: param,
		comms:  comms,
		sender: sender,
		events: events,
		c:      make(chan roundRequest, param.HistoricalRoundsBufferLen),
	}
}

// LookupHistoricalRound sends the lookup request to the internal handler and
// returns the result on the callback.
func (m *manager) LookupHistoricalRound(
	rid id.Round, callback RoundResultCallback) error {
	if rid == 0 {
		return errors.New("Cannot look up round 0, rounds start at 1")
	}

	select {
	case m.c <- roundRequest{rid, callback, 0}:
		return nil
	default:
		return errors.Errorf("Cannot look up round %d, channel is full", rid)
	}
}

// StartProcesses starts the Long running thread that process historical rounds.
// The thread can be killed by sending a signal to the returned stoppable.
func (m *manager) StartProcesses() *stoppable.Single {
	stop := stoppable.NewSingle("TrackNetwork")
	go m.processHistoricalRounds(m.comms, stop)
	return stop
}

// processHistoricalRounds is a long-running thread that process historical
// rounds. The thread can be killed by triggering the stoppable. It takes a
// comms interface to aid in testing.
func (m *manager) processHistoricalRounds(comm Comms, stop *stoppable.Single) {
	timerCh := make(<-chan time.Time)
	var roundRequests []roundRequest

	for {
		shouldProcess := false

		// Wait for a quit or new round to check
		select {
		case <-stop.Quit():
			// Return all roundRequests in the queue to the input channel so
			// that they can be checked in the future. If the queue is full,
			// then disable them as processing so that they are picked up from
			// the beginning.
			for _, r := range roundRequests {
				select {
				case m.c <- r:
				default:
				}
			}

			stop.ToStopped()
			return

		case <-timerCh:
			// If the timer elapses, then process roundRequests to ensure that
			// the delay is not too long
			if len(roundRequests) > 0 {
				shouldProcess = true
			}

		case r := <-m.c:
			// Get new round to look up and force a lookup if
			jww.DEBUG.Printf("Received and queueing round %d for historical "+
				"rounds lookup.", r.rid)

			roundRequests = append(roundRequests, r)

			if len(roundRequests) > int(m.params.MaxHistoricalRounds) {
				shouldProcess = true
			} else if len(roundRequests) != 0 {
				// If this is the first round, start the timeout
				timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			}
		}

		if !shouldProcess {
			continue
		}

		rounds := make([]uint64, len(roundRequests))
		for i, rr := range roundRequests {
			rounds[i] = uint64(rr.rid)
		}

		// Send the historical roundRequests request
		hr := &pb.HistoricalRounds{Rounds: rounds}

		var gwHost *connect.Host
		result, err := m.sender.SendToAny(
			func(host *connect.Host) (interface{}, error) {
				jww.DEBUG.Printf("Requesting Historical rounds %v from "+
					"gateway %s", rounds, host.GetId())

				gwHost = host

				return comm.RequestHistoricalRounds(host, hr)
			}, stop)

		if err != nil {
			jww.ERROR.Printf("Failed to request historical roundRequests "+
				"data for rounds %v: %s", rounds, err)

			// If the check fails to resolve, then break the loop so that they
			// will be checked again
			timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			continue
		}

		response := result.(*pb.HistoricalRoundsResponse)

		rids, retries := processHistoricalRoundsResponse(response,
			roundRequests, m.params.MaxHistoricalRoundsRetries, m.events)

		m.events.Report(1, "HistoricalRounds", "Metrics",
			fmt.Sprintf("Received %d historical rounds from gateway %s: %v",
				len(response.Rounds), gwHost, rids))

		// Reset the buffer to those left to retry now that all have been
		// checked
		roundRequests = retries

		// Now reset the timer, this prevents immediate reprocessing of the
		// retries, limiting it to the next historical round request when buffer
		// is full OR next timer tick
		timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
	}
}

func processHistoricalRoundsResponse(response *pb.HistoricalRoundsResponse,
	roundRequests []roundRequest, maxRetries uint, events event.Manager) (
	[]uint64, []roundRequest) {
	retries := make([]roundRequest, 0)
	rids := make([]uint64, 0)

	// Process the returned historical roundRequests
	for i, roundInfo := range response.Rounds {
		// The interface has missing returns returned as nil, such roundRequests
		// need to be removed as processing so that the network follower will
		// pick them up in the future.
		if roundInfo == nil {
			var errMsg string
			roundRequests[i].numAttempts++

			if roundRequests[i].numAttempts == maxRetries {
				errMsg = fmt.Sprintf("Failed to retrieve historical round %d "+
					"on last attempt, will not try again", roundRequests[i].rid)
				go roundRequests[i].RoundResultCallback(Round{}, false)
			} else {
				retries = append(retries, roundRequests[i])
				errMsg = fmt.Sprintf("Failed to retrieve historical round "+
					"%d, will try up to %d more times", roundRequests[i].rid,
					maxRetries-roundRequests[i].numAttempts)
			}

			jww.WARN.Printf(errMsg)
			events.Report(5, "HistoricalRounds", "Error", errMsg)
			continue
		}

		// Successfully retrieved roundRequests are returned on the callback
		go roundRequests[i].RoundResultCallback(MakeRound(roundInfo), true)

		rids = append(rids, roundInfo.ID)
	}

	return rids, retries
}
