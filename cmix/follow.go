////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

// follow.go tracks the network for:
//   1. The status of the network and its accessibility
//   2. New/Active/Complete rounds and their contact gateways
//   3. Node addition and removal
// This information is tracked by polling a gateway for the network definition
// file (NDF). Once it detects an event it sends it off to the proper channel
// for a worker to update the client state (add/remove a nodes, check for
// messages at a gateway, etc). See:
//   - /nodes/register.go for add/remove nodes events
//   - /rounds/historical.go for old round retrieval
//   - /rounds/retrieve.go for message retrieval
//   - /message/handle.go decryption, partitioning, and signaling of messages
//   - /health/tracker.go - tracks the state of the network through the network
//		instance

import (
	"crypto/hmac"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/xx_network/primitives/ndf"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID/store"
	"gitlab.com/elixxir/client/v4/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	debugTrackPeriod = 1 * time.Minute

	// Estimate the number of rounds per second in the network. Will need
	// updated someday in order to correctly determine how far back to search
	// rounds for messages as the network continues to grow, otherwise message
	// drops occur.
	estimatedRoundsPerSecond = 5
)

// followNetworkComms is a comms interface to make testing easier.
type followNetworkComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendPoll(host *connect.Host, message *pb.GatewayPoll) (
		*pb.GatewayPollResponse, time.Time, time.Duration, error)
	RequestMessages(host *connect.Host, message *pb.GetMessages) (
		*pb.GetMessagesResponse, error)
}

// followNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func (c *client) followNetwork(report ClientErrorReport,
	stop *stoppable.Single) {
	ticker := time.NewTicker(c.param.TrackNetworkPeriod)
	TrackTicker := time.NewTicker(debugTrackPeriod)
	rng := c.rng.GetStream()

	// abandon tracks rounds which data was not found out about in
	// the verbose rounds debugging mode
	abandon := func(round id.Round) { return }
	dummyAbandon := func(round id.Round) { return }
	if c.verboseRounds != nil {
		abandon = func(round id.Round) {
			c.verboseRounds.denote(round, Abandoned)
		}
	}

	for {
		select {
		case <-stop.Quit():
			rng.Close()
			stop.ToStopped()
			return
		case <-ticker.C:
			operator := func(toTrack []receptionID.IdentityUse) error {

				// set up tracking tools
				wg := &sync.WaitGroup{}
				wg.Add(len(toTrack))

				// trigger the first separately because it will get network state
				// updates
				go func() {
					c.follow(toTrack[0], report, rng, c.comms, stop, abandon,
						true)
					wg.Done()
				}()

				//trigger all others without getting network state updates
				for i := 1; i < len(toTrack); i++ {
					go func(index int) {
						c.follow(toTrack[index], report, rng, c.comms, stop,
							dummyAbandon, false)
						wg.Done()
					}(i)
				}

				//wait for all to complete
				wg.Wait()
				return nil
			}

			//denote the execution
			atomic.AddUint64(c.tracker, 1)

			// track the message on every identity
			stream := c.rng.GetStream()
			err := c.Tracker.ForEach(
				int(c.param.MaxParallelIdentityTracks),
				stream,
				c.Space.GetAddressSpaceWithoutWait(),
				operator)
			stream.Close()

			//update clock skew
			estimatedSkew := c.skewTracker.Aggregate()
			// invert the skew because we need to reverse it
			netTime.SetOffset(-estimatedSkew)

			if err != nil {
				jww.ERROR.Printf("failed to operate on identities to "+
					"track: %s", err)
				continue
			}

		case <-TrackTicker.C:
			numPolls := atomic.SwapUint64(c.tracker, 0)
			if c.numLatencies != 0 {
				latencyAvg := time.Nanosecond * time.Duration(
					c.latencySum/c.numLatencies)
				c.latencySum, c.numLatencies = 0, 0

				infoMsg := fmt.Sprintf("Polled the network %d times in the "+
					"last %s, with an average newest packet latency of %s",
					numPolls, debugTrackPeriod, latencyAvg)

				jww.INFO.Printf(infoMsg)
				c.events.Report(1, "Polling", "MetricsWithLatency", infoMsg)
			} else {
				infoMsg := fmt.Sprintf(
					"Polled the network %d times in the last %s", numPolls,
					debugTrackPeriod)

				jww.INFO.Printf(infoMsg)
				c.events.Report(1, "Polling", "Metrics", infoMsg)
			}
		}
	}
}

// follow executes an iteration of the follower for a specific identity
func (c *client) follow(identity receptionID.IdentityUse,
	report ClientErrorReport, rng csprng.Source, comms followNetworkComms,
	stop *stoppable.Single, abandon func(round id.Round), getUpdates bool) {

	// While polling with a fake identity, it is necessary to have populated
	// earliestRound data. However, as with fake identities, we want the values
	// to be randomly generated rather than based on actual state.
	if identity.Fake {
		fakeEr := &store.EarliestRound{}
		fakeEr.Set(c.getFakeEarliestRound())
		identity.ER = fakeEr
	}

	// Get client version for poll
	version := c.session.GetClientVersion()

	// Poll network updates
	pollReq := pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: c.instance.GetPartialNdf().GetHash(),
		},
		LastUpdate:     uint64(c.instance.GetLastUpdateID()),
		ReceptionID:    identity.EphId[:],
		StartTimestamp: identity.StartValid.UnixNano(),
		EndTimestamp:   identity.EndValid.UnixNano(),
		ClientVersion:  []byte(version.String()),
		FastPolling:    c.param.FastPolling,
		LastRound:      uint64(identity.ER.Get()),
		DisableUpdates: !getUpdates,
	}

	var rtt time.Duration
	var sendTo *id.ID
	var startTime time.Time

	result, err := c.SendToAny(func(host *connect.Host) (interface{}, error) {
		jww.DEBUG.Printf("Executing poll for %v(%s) range: %s-%s(%s) from %s",
			identity.EphId.Int64(), identity.Source, identity.StartValid,
			identity.EndValid, identity.EndValid.Sub(identity.StartValid),
			host.GetId())
		var err error
		var response *pb.GatewayPollResponse
		response, startTime, rtt, err = comms.SendPoll(host, &pollReq)
		sendTo = host.GetId()
		return response, err
	}, stop)

	// Exit if the thread has been stopped
	if stoppable.CheckErr(err) {
		jww.INFO.Print(err)
		return
	}

	now := netTime.Now()

	if err != nil {
		if report != nil {
			report(
				"NetworkFollower",
				fmt.Sprintf("Failed to poll network, \"%s\":", err.Error()),
				fmt.Sprintf("%+v", err),
			)
		}
		errMsg := fmt.Sprintf("Unable to poll gateway: %+v", err)
		c.events.Report(10, "Polling", "Error", errMsg)
		jww.ERROR.Print(errMsg)
		return
	}

	pollResp := result.(*pb.GatewayPollResponse)

	//execute clock skew update
	c.skewTracker.Add(sendTo, startTime,
		time.Unix(0, pollResp.ReceivedTs),
		rtt, time.Duration(pollResp.GatewayDelay))

	// ---- Process Network State Update Data ----
	gwRoundsState := &knownRounds.KnownRounds{}
	err = gwRoundsState.Unmarshal(pollResp.KnownRounds)
	if err != nil {
		jww.ERROR.Printf("Failed to unmarshal: %+v", err)
		return
	}

	// ---- Node Events ----
	// NOTE: this updates the structure, AND sends events over the nodes update
	//       channels about new and removed nodes
	if pollResp.PartialNDF != nil {
		err = c.instance.UpdatePartialNdf(pollResp.PartialNDF)
		if err != nil {
			jww.ERROR.Printf("Unable to update partial NDF: %+v", err)
			return
		}

		//set the number of nodes
		numNodes := uint64(0)
		for _, n := range c.instance.GetPartialNdf().Get().Nodes {
			if n.Status != ndf.Stale {
				numNodes++
			}
		}
		atomic.StoreUint64(c.numNodes, numNodes)

		// update gateway connections
		c.UpdateNdf(c.GetInstance().GetPartialNdf().Get())
		c.session.SetNDF(c.GetInstance().GetPartialNdf().Get())
	}

	// Update the address space size
	if len(c.instance.GetPartialNdf().Get().AddressSpace) != 0 {
		c.UpdateAddressSpace(c.instance.GetPartialNdf().Get().AddressSpace[0].Size)
	}

	// NOTE: this updates rounds and updates the tracking of the health of the
	// network
	if pollResp.Updates != nil {
		// TODO: ClientErr needs to know the source of the error and it doesn't yet
		// Iterate over ClientErrors for each RoundUpdate
		for _, update := range pollResp.Updates {

			// Ignore irrelevant updates
			if update.State != uint32(states.COMPLETED) &&
				update.State != uint32(states.FAILED) {
				continue
			}

			marshaledTid := c.session.GetTransmissionID().Marshal()
			for _, clientErr := range update.ClientErrors {
				// If this ClientId appears in the ClientError
				if hmac.Equal(clientErr.ClientId, marshaledTid) {

					// Obtain relevant NodeGateway information
					nid, err := id.Unmarshal(clientErr.Source)
					if err != nil {
						jww.ERROR.Printf("Unable to get NodeID: %+v", err)
						return
					}

					// Mutate the update to indicate failure due to a ClientError
					// FIXME: Should be able to trigger proper type of round
					//  event without mutating the RoundInfo. Signature also
					//  needs verified before keys are deleted.
					update.State = uint32(states.FAILED)

					// trigger a reregistration with the node
					c.Registrar.TriggerNodeRegistration(nid)
				}
			}
		}

		// Trigger RoundEvents for all polled updates, including modified rounds
		// with ClientErrors
		err = c.instance.RoundUpdates(pollResp.Updates)
		if err != nil {
			jww.ERROR.Printf("%+v", err)
			return
		}

		newestTS := uint64(0)
		for i := 0; i < len(pollResp.Updates[len(pollResp.Updates)-1].Timestamps); i++ {
			if pollResp.Updates[len(pollResp.Updates)-1].Timestamps[i] != 0 {
				newestTS = pollResp.Updates[len(pollResp.Updates)-1].Timestamps[i]
			}
		}

		newest := time.Unix(0, int64(newestTS))

		if newest.After(now) {
			deltaDur := newest.Sub(now)
			c.latencySum = uint64(deltaDur)
			c.numLatencies++
		}
	}

	// ---- Identity Specific Round Processing -----
	if identity.Fake {
		jww.DEBUG.Printf("Not processing result, identity.Fake == true")
		return
	}

	if len(pollResp.Filters.Filters) == 0 {
		jww.TRACE.Printf("No filters found for the passed ID %d (%s), "+
			"skipping processing.", identity.EphId.Int64(), identity.Source)
		return
	}

	// Prepare the filter objects for processing
	filterList := make([]*RemoteFilter, 0, len(pollResp.Filters.Filters))
	for i := range pollResp.Filters.Filters {
		if len(pollResp.Filters.Filters[i].Filter) != 0 {
			filterList = append(filterList,
				NewRemoteFilter(pollResp.Filters.Filters[i]))
		}
	}

	// Check rounds using the round checker function, which determines if there
	// are messages waiting in rounds and then sends signals to the appropriate
	// handling threads
	roundChecker := func(rid id.Round) bool {
		hasMessage := Checker(rid, filterList, identity.CR)
		if !hasMessage && c.verboseRounds != nil {
			c.verboseRounds.denote(rid, RoundState(NoMessageAvailable))
		}
		//jww.INFO.Printf("[LOOKUP] round %d checked for %d, has message: %v", rid, identity.EphId.Int64(), hasMessage)
		return hasMessage
	}

	// Move the earliest unknown round tracker forward to the earliest tracked
	// round if it is behind
	earliestTrackedRound := id.Round(pollResp.EarliestRound)
	c.SetFakeEarliestRound(earliestTrackedRound)
	updatedEarliestRound, _, _ := identity.ER.Set(earliestTrackedRound)
	/* this code looks to be legacy, commenting out to disable and see what happens
	// If there was no registered rounds for the identity
	if old == 0 {
		lastCheckedRound := gwRoundsState.GetLastChecked()
		// Approximate the earliest possible round that messages could be
		// received on this ID by using an estimate of how many rounds the
		// network runs per second
		timeSinceStartValid := netTime.Now().Sub(identity.StartValid)
		roundsDelta :=
			uint(timeSinceStartValid / time.Second * estimatedRoundsPerSecond)
		if roundsDelta < c.param.KnownRoundsThreshold {
			roundsDelta = c.param.KnownRoundsThreshold
		}

		if id.Round(roundsDelta) > lastCheckedRound {
			// Handles edge case for new networks to prevent starting at
			// negative rounds
			updatedEarliestRound = 1
		} else {
			updatedEarliestRound = lastCheckedRound - id.Round(roundsDelta)
			earliestFilterRound := filterList[0].FirstRound() // Length of filterList always > 0

			// If the network appears to be moving faster than our estimate,
			// causing earliestFilterRound to be lower, we will instead use the
			// earliestFilterRound, which will ensure messages are not dropped
			// as long as contacted gateway has all data
			if updatedEarliestRound > earliestFilterRound {
				updatedEarliestRound = earliestFilterRound
			}
		}
		identity.ER.Set(updatedEarliestRound)
	}*/

	// Loop through all rounds the client does not know about and the gateway
	// does, checking the bloom filter for the user to see if there are messages
	// for the user (bloom not implemented yet)
	// Threshold is the earliest round that will not be excluded from earliest
	// remaining
	earliestRemaining, roundsWithMessages, roundsUnknown :=
		gwRoundsState.RangeUnchecked(
			updatedEarliestRound, c.param.KnownRoundsThreshold, roundChecker, 100)

	jww.DEBUG.Printf("Processed RangeUnchecked for %d, Oldest: %d, "+
		"firstUnchecked: %d, last Checked: %d, threshold: %d, "+
		"NewEarliestRemaining: %d, NumWithMessages: %d, NumUnknown: %d",
		identity.EphId.Int64(), updatedEarliestRound, gwRoundsState.GetFirstUnchecked(),
		gwRoundsState.GetLastChecked(), c.param.KnownRoundsThreshold,
		earliestRemaining, len(roundsWithMessages), len(roundsUnknown))

	_, _, changed := identity.ER.Set(earliestRemaining)
	if changed {
		jww.TRACE.Printf("External returns of RangeUnchecked: %d, %v, %v",
			earliestRemaining, roundsWithMessages, roundsUnknown)
		jww.DEBUG.Printf("New Earliest Remaining: %d, Gateways last checked: %d",
			earliestRemaining, gwRoundsState.GetLastChecked())
	}

	var roundsWithMessages2 []id.Round

	roundsWithMessages2 = identity.UR.Iterate(func(rid id.Round) bool {
		if gwRoundsState.Checked(rid) {
			return Checker(rid, filterList, identity.CR)
		}
		return false
	}, roundsUnknown, abandon)

	for i := 0; i < len(roundsWithMessages); i++ {
		rid := roundsWithMessages[i]
		// Denote that the round has been looked at in the tracking store
		if identity.CR.Check(rid) {
			c.GetMessagesFromRound(rid, identity.EphemeralIdentity)
		}
	}

	identity.CR.Prune()
	err = identity.CR.SaveCheckedRounds()
	if err != nil {
		jww.ERROR.Printf("Could not save rounds for identity %d (%s): %+v",
			identity.EphId.Int64(), identity.Source, err)
	}

	for _, rid := range roundsWithMessages2 {
		c.GetMessagesFromRound(rid, identity.EphemeralIdentity)
	}

	if c.verboseRounds != nil {
		trackingStart := updatedEarliestRound
		if uint(earliestRemaining-updatedEarliestRound) > c.param.KnownRoundsThreshold {
			trackingStart = earliestRemaining - id.Round(c.param.KnownRoundsThreshold)
		}

		jww.DEBUG.Printf("Rounds tracked: %v to %v", trackingStart, earliestRemaining)

		for i := trackingStart; i <= earliestRemaining; i++ {
			state := Unchecked
			for _, rid := range roundsWithMessages {
				if rid == i {
					state = MessageAvailable
				}
			}
			for _, rid := range roundsWithMessages2 {
				if rid == i {
					state = MessageAvailable
				}
			}
			for _, rid := range roundsUnknown {
				if rid == i {
					state = Unknown
				}
			}
			c.verboseRounds.denote(i, RoundState(state))
		}
	}

}

// getFakeEarliestRound generates a random earliest round for a fake identity.
func (c *client) getFakeEarliestRound() id.Round {
	rng := c.rng.GetStream()
	b, err := csprng.Generate(8, rng)
	if err != nil {
		jww.FATAL.Panicf("Could not get random number: %v", err)
	}
	rng.Close()

	rangeVal := binary.LittleEndian.Uint64(b) % 800

	earliestKnown := atomic.LoadUint64(c.earliestRound)

	return id.Round(earliestKnown - rangeVal)
}
