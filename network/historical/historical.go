///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package historical

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Historical Rounds looks up the round history via random gateways.
// It batches these quests but never waits longer than
// params.HistoricalRoundsPeriod to do a lookup.
// Historical rounds receives input from:
//   - Network Follower (/network/follow.go)
// Historical Rounds sends the output to:
//	 - Message Retrieval Workers (/network/round/retrieve.go)

type Retriever interface {
	StartProcessies() *stoppable.Single
	LookupHistoricalRound(rid id.Round, callback RoundResultCallback) error
}

// manager is the controlling structure
type manager struct {
	params params.Historical

	comms  RoundsComms
	sender gateway.Sender
	events interfaces.EventManager

	c chan roundRequest
}

//RoundsComms interface to increase east of testing of historical rounds
type RoundsComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	RequestHistoricalRounds(host *connect.Host,
		message *pb.HistoricalRounds) (*pb.HistoricalRoundsResponse, error)
}

//RoundResultCallback is the used callback when a round is found
type RoundResultCallback func(info *pb.RoundInfo, success bool)

//roundRequest is an internal structure which tracks a request
type roundRequest struct {
	rid id.Round
	RoundResultCallback
	numAttempts uint
}

func NewRetriever(param params.Historical, comms RoundsComms,
	sender gateway.Sender, events interfaces.EventManager) Retriever {
	return &manager{
		params: param,
		comms:  comms,
		sender: sender,
		events: events,
		c:      make(chan roundRequest, param.HistoricalRoundsBufferLen),
	}
}

// LookupHistoricalRound sends the lookup request to the internal handler
// and will return the result on the callback when it returns
func (m *manager) LookupHistoricalRound(rid id.Round, callback RoundResultCallback) error {
	if rid == 0 {
		return errors.Errorf("Cannot lookup round 0, rounds start at 1")
	}
	select {
	case m.c <- roundRequest{
		rid:                 rid,
		RoundResultCallback: callback,
		numAttempts:         0,
	}:
		return nil
	default:
		return errors.Errorf("Cannot lookup round %d, "+
			"channel is full", rid)
	}
}

// StartProcessies starts the Long running thread which
// process historical rounds. Can be killed by sending a
// signal to the quit channel
func (m *manager) StartProcessies() *stoppable.Single {
	stop := stoppable.NewSingle("TrackNetwork")
	go m.processHistoricalRounds(m.comms, stop)
	return stop
}

// processHistoricalRounds is a long running thread which
// process historical rounds. Can be killed by sending
// a signal to the quit channel takes a comms interface to aid in testing
func (m *manager) processHistoricalRounds(comm RoundsComms, stop *stoppable.Single) {

	timerCh := make(<-chan time.Time)

	var roundRequests []roundRequest

	for {

		shouldProcess := false
		// wait for a quit or new round to check
		select {
		case <-stop.Quit():
			// return all roundRequests in the queue to the input channel so they can
			// be checked in the future. If the queue is full, disable them as
			// processing so they are picked up from the beginning
			for _, r := range roundRequests {
				select {
				case m.c <- r:
				default:
				}
			}
			stop.ToStopped()
			return
		// if the timer elapses process roundRequests to ensure the delay isn't too long
		case <-timerCh:
			if len(roundRequests) > 0 {
				shouldProcess = true
			}
		// get new round to lookup and force a lookup if
		case r := <-m.c:
			jww.DEBUG.Printf("Received and queueing round %d for "+
				"historical rounds lookup", r.rid)
			roundRequests = append(roundRequests, r)
			if len(roundRequests) > int(m.params.MaxHistoricalRounds) {
				shouldProcess = true
			} else if len(roundRequests) != 0 {
				//if this is the first round, start the timeout
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

		//send the historical roundRequests request
		hr := &pb.HistoricalRounds{
			Rounds: rounds,
		}

		var gwHost *connect.Host
		result, err := m.sender.SendToAny(func(host *connect.Host) (interface{}, error) {
			jww.DEBUG.Printf("Requesting Historical rounds %v from "+
				"gateway %s", rounds, host.GetId())
			gwHost = host
			return comm.RequestHistoricalRounds(host, hr)
		}, stop)

		if err != nil {
			jww.ERROR.Printf("Failed to request historical roundRequests "+
				"data for rounds %v: %s", rounds, err)
			// if the check fails to resolve, break the loop and so they will be
			// checked again
			timerCh = time.NewTimer(m.params.HistoricalRoundsPeriod).C
			continue
		}
		response := result.(*pb.HistoricalRoundsResponse)

		rids := make([]uint64, 0)
		// process the returned historical roundRequests.
		for i, roundInfo := range response.Rounds {
			// The interface has missing returns returned as nil, such roundRequests
			// need be be removes as processing so the network follower will
			// pick them up in the future.
			if roundInfo == nil {
				var errMsg string
				roundRequests[i].numAttempts++
				if roundRequests[i].numAttempts == m.params.MaxHistoricalRoundsRetries {
					errMsg = fmt.Sprintf("Failed to retreive historical "+
						"round %d on last attempt, will not try again",
						roundRequests[i].rid)
					go roundRequests[i].RoundResultCallback(nil, false)
				} else {
					select {
					case m.c <- roundRequests[i]:
						errMsg = fmt.Sprintf("Failed to retreive historical "+
							"round %d, will try up to %d more times",
							roundRequests[i].rid, m.params.MaxHistoricalRoundsRetries-roundRequests[i].numAttempts)
					default:
						errMsg = fmt.Sprintf("Failed to retreive historical "+
							"round %d, failed to try again, round will not be "+
							"retreived", roundRequests[i].rid)
					}
				}
				jww.WARN.Printf(errMsg)
				m.events.Report(5, "HistoricalRounds",
					"Error", errMsg)
				continue
			}
			// Successfully retrieved roundRequests are returned on the callback
			go roundRequests[i].RoundResultCallback(roundInfo, true)

			rids = append(rids, roundInfo.ID)
		}

		m.events.Report(1, "HistoricalRounds", "Metrics",
			fmt.Sprintf("Received %d historical rounds from"+
				" gateway %s: %v", len(response.Rounds), gwHost,
				rids))

		//clear the buffer now that all have been checked
		roundRequests = make([]roundRequest, 0)
	}
}
