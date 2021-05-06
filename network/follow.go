///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package network

// follow.go tracks the network for:
//   1. The status of the network and its accessibility
//   2. New/Active/Complete rounds and their contact gateways
//   3. Node addition and removal
// This information is tracked by polling a gateway for the network definition
// file (NDF). Once it detects an event it sends it off to the proper channel
// for a worker to update the client state (add/remove a node, check for
// messages at a gateway, etc). See:
//   - /node/register.go for add/remove node events
//   - /rounds/historical.go for old round retrieval
//   - /rounds/retrieve.go for message retrieval
//   - /message/handle.go decryption, partitioning, and signaling of messages
//   - /health/tracker.go - tracks the state of the network through the network
//		instance

import (
	"bytes"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/network/rounds"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/knownRounds"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"sync/atomic"
	"time"
)

const debugTrackPeriod = 1 * time.Minute

//comms interface makes testing easier
type followNetworkComms interface {
	GetHost(hostId *id.ID) (*connect.Host, bool)
	SendPoll(host *connect.Host, message *pb.GatewayPoll) (*pb.GatewayPollResponse, error)
}

// followNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func (m *manager) followNetwork(report interfaces.ClientErrorReport, quitCh <-chan struct{}) {
	ticker := time.NewTicker(m.param.TrackNetworkPeriod)
	TrackTicker := time.NewTicker(debugTrackPeriod)
	rng := m.Rng.GetStream()

	done := false
	for !done {
		select {
		case <-quitCh:
			rng.Close()
			done = true
		case <-ticker.C:
			m.follow(report, rng, m.Comms)
		case <-TrackTicker.C:
			numPolls := atomic.SwapUint64(m.tracker, 0)
			jww.INFO.Printf("Polled the network %d times in the "+
				"last %s", numPolls, debugTrackPeriod)
		}
	}
}

// executes each iteration of the follower
func (m *manager) follow(report interfaces.ClientErrorReport, rng csprng.Source, comms followNetworkComms) {

	//get the identity we will poll for
	identity, err := m.Session.Reception().GetIdentity(rng)
	if err != nil {
		jww.FATAL.Panicf("Failed to get an identity, this should be "+
			"impossible: %+v", err)
	}

	atomic.AddUint64(m.tracker, 1)

	// Get client version for poll
	version := m.Session.GetClientVersion()

	// Poll network updates
	pollReq := pb.GatewayPoll{
		Partial: &pb.NDFHash{
			Hash: m.Instance.GetPartialNdf().GetHash(),
		},
		LastUpdate:     uint64(m.Instance.GetLastUpdateID()),
		ReceptionID:    identity.EphId[:],
		StartTimestamp: identity.StartRequest.UnixNano(),
		EndTimestamp:   identity.EndRequest.UnixNano(),
		ClientVersion:  []byte(version.String()),
	}

	result, err := m.GetSender().SendToAny(func(host *connect.Host) (interface{}, error) {
		jww.DEBUG.Printf("Executing poll for %v(%s) range: %s-%s(%s) from %s",
			identity.EphId.Int64(), identity.Source, identity.StartRequest,
			identity.EndRequest, identity.EndRequest.Sub(identity.StartRequest), host.GetId())
		result, err := comms.SendPoll(host, &pollReq)
		if err != nil {
			if report != nil {
				report(
					"NetworkFollower",
					fmt.Sprintf("Failed to poll network, \"%s\", Gateway: %s", err.Error(), host.String()),
					fmt.Sprintf("%+v", err),
				)
			}
			jww.ERROR.Printf("Unable to poll %s for NDF: %+v", host, err)
		}
		return result, err
	})
	if err != nil {
		return
	}

	pollResp := result.(*pb.GatewayPollResponse)

	// ---- Process Network State Update Data ----
	gwRoundsState := &knownRounds.KnownRounds{}
	err = gwRoundsState.Unmarshal(pollResp.KnownRounds)
	if err != nil {
		jww.ERROR.Printf("Failed to unmarshal: %+v", err)
		return
	}

	// ---- Node Events ----
	// NOTE: this updates the structure, AND sends events over the node
	//       update channels about new and removed nodes
	if pollResp.PartialNDF != nil {
		err = m.Instance.UpdatePartialNdf(pollResp.PartialNDF)
		if err != nil {
			jww.ERROR.Printf("Unable to update partial NDF: %+v", err)
			return
		}

		// update gateway connections
		m.GetSender().UpdateNdf(m.GetInstance().GetPartialNdf().Get())
	}

	//check that the stored address space is correct
	m.Session.Reception().UpdateIdSize(uint(m.Instance.GetPartialNdf().Get().AddressSpaceSize))
	// Updates any id size readers of a network compliant id size
	m.Session.Reception().MarkIdSizeAsSet()
	// NOTE: this updates rounds and updates the tracking of the health of the
	// network
	if pollResp.Updates != nil {
		err = m.Instance.RoundUpdates(pollResp.Updates)
		if err != nil {
			jww.ERROR.Printf("%+v", err)
			return
		}

		// TODO: ClientErr needs to know the source of the error and it doesn't yet
		// Iterate over ClientErrors for each RoundUpdate
		for _, update := range pollResp.Updates {

			// Ignore irrelevant updates
			if update.State != uint32(states.COMPLETED) && update.State != uint32(states.FAILED) {
				continue
			}

			for _, clientErr := range update.ClientErrors {
				// If this Client appears in the ClientError
				if bytes.Equal(clientErr.ClientId, m.Session.GetUser().TransmissionID.Marshal()) {

					// Obtain relevant NodeGateway information
					nid, err := id.Unmarshal(clientErr.Source)
					if err != nil {
						jww.ERROR.Printf("Unable to get NodeID: %+v", err)
						return
					}
					nGw, err := m.Instance.GetNodeAndGateway(nid)
					if err != nil {
						jww.ERROR.Printf("Unable to get gateway: %+v", err)
						return
					}

					// FIXME: Should be able to trigger proper type of round event
					// FIXME: without mutating the RoundInfo. Signature also needs verified
					// FIXME: before keys are deleted
					update.State = uint32(states.FAILED)
					rnd, err := m.Instance.GetWrappedRound(id.Round(update.ID))
					if err != nil {
						jww.ERROR.Printf("Failed to report client error: "+
							"Could not get round for event triggering: "+
							"Unable to get round %d from instance: %+v",
							id.Round(update.ID), err)
						break
					}
					m.Instance.GetRoundEvents().TriggerRoundEvent(rnd)

					// delete all existing keys and trigger a re-registration with the relevant Node
					m.Session.Cmix().Remove(nid)
					m.Instance.GetAddGatewayChan() <- nGw
				}
			}
		}
	}

	// ---- Identity Specific Round Processing -----
	if identity.Fake {
		jww.DEBUG.Printf("not processing result, identity.Fake == true")
		return
	}

	if len(pollResp.Filters.Filters) == 0 {
		jww.TRACE.Printf("No filters found for the passed ID %d (%s), "+
			"skipping processing.", identity.EphId.Int64(), identity.Source)
		return
	}

	//get the range fo filters which are valid for the identity
	filtersStart, filtersEnd, outOfBounds := rounds.ValidFilterRange(identity, pollResp.Filters)

	//check if there are any valid filters returned
	if outOfBounds {
		return
	}

	//prepare the filter objects for processing
	filterList := make([]*rounds.RemoteFilter, 0, filtersEnd-filtersStart)
	for i := filtersStart; i < filtersEnd; i++ {
		if len(pollResp.Filters.Filters[i].Filter) != 0 {
			filterList = append(filterList, rounds.NewRemoteFilter(pollResp.Filters.Filters[i]))
		}
	}

	// check rounds using the round checker function which determines if there
	// are messages waiting in rounds and then sends signals to the appropriate
	// handling threads
	roundChecker := func(rid id.Round) bool {
		return rounds.Checker(rid, filterList, identity.CR)
	}

	// move the earliest unknown round tracker forward to the earliest
	// tracked round if it is behind
	earliestTrackedRound := id.Round(pollResp.EarliestRound)
	updated, _ := identity.ER.Set(earliestTrackedRound)

	// loop through all rounds the client does not know about and the gateway
	// does, checking the bloom filter for the user to see if there are
	// messages for the user (bloom not implemented yet)
	//threshold is the earliest round that will not be excluded from earliest remaining
	earliestRemaining, roundsWithMessages, roundsUnknown := gwRoundsState.RangeUnchecked(updated,
		m.param.KnownRoundsThreshold, roundChecker)
	_, changed := identity.ER.Set(earliestRemaining)
	if changed {
		jww.TRACE.Printf("External returns of RangeUnchecked: %d, %v, %v", earliestRemaining, roundsWithMessages, roundsUnknown)
		jww.DEBUG.Printf("New Earliest Remaining: %d", earliestRemaining)
	}

	roundsWithMessages2 := identity.UR.Iterate(func(rid id.Round) bool {
		if gwRoundsState.Checked(rid) {
			return rounds.Checker(rid, filterList, identity.CR)
		}
		return false
	}, roundsUnknown)

	for _, rid := range roundsWithMessages {
		if identity.CR.Check(rid) {
			m.round.GetMessagesFromRound(rid, identity)
		}
	}

	identity.CR.Prune()
	err = identity.CR.SaveCheckedRounds()
	if err != nil {
		jww.ERROR.Printf("Could not save rounds for identity %d (%s): %+v",
			identity.EphId.Int64(), identity.Source, err)
	}

	for _, rid := range roundsWithMessages2 {
		m.round.GetMessagesFromRound(rid, identity)
	}
}
