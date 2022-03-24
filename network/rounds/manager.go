///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/historical"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/network/rounds/store"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
)

type Pickup interface {
	StartProcessors() stoppable.Stoppable
	GetMessagesFromRound(roundID id.Round, identity interfaces.EphemeralIdentity)
}

type manager struct {
	params  params.Rounds
	sender  gateway.Sender
	session *storage.Session

	comms MessageRetrievalComms

	historical historical.Retriever

	rng *fastRNG.StreamGenerator

	instance RoundGetter

	lookupRoundMessages chan roundLookup
	messageBundles      chan<- message.Bundle

	unchecked *store.UncheckedRoundStore
}

func NewPickup(params params.Rounds, bundles chan<- message.Bundle,
	sender gateway.Sender, historical historical.Retriever, rng *fastRNG.StreamGenerator,
	instance RoundGetter, kv *versioned.KV) Pickup {
	unchecked := store.NewOrLoadUncheckedStore(kv)
	m := &manager{
		params:              params,
		lookupRoundMessages: make(chan roundLookup, params.LookupRoundsBufferLen),
		messageBundles:      bundles,
		sender:              sender,
		historical:          historical,
		rng:                 rng,
		instance:            instance,
		unchecked:           unchecked,
	}

	return m
}

func (m *manager) StartProcessors() stoppable.Stoppable {

	multi := stoppable.NewMulti("Rounds")

	//start the historical rounds thread

	//start the message retrieval worker pool
	for i := uint(0); i < m.params.NumMessageRetrievalWorkers; i++ {
		stopper := stoppable.NewSingle("Message Retriever " + strconv.Itoa(int(i)))
		go m.processMessageRetrieval(m.comms, stopper)
		multi.Add(stopper)
	}

	// Start the periodic unchecked round worker
	if !m.params.RealtimeOnly {
		stopper := stoppable.NewSingle("UncheckRound")
		go m.processUncheckedRounds(m.params.UncheckRoundPeriod, backOffTable, stopper)
		multi.Add(stopper)
	}

	return multi
}
