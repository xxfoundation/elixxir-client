///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package rounds

import (
	"fmt"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/stoppable"
)

type Manager struct {
	params params.Rounds

	internal.Internal
	sender *gateway.Sender

	historicalRounds    chan historicalRoundRequest
	lookupRoundMessages chan roundLookup
	messageBundles      chan<- message.Bundle
}

func NewManager(internal internal.Internal, params params.Rounds,
	bundles chan<- message.Bundle, sender *gateway.Sender) *Manager {
	m := &Manager{
		params: params,

		historicalRounds:    make(chan historicalRoundRequest, params.HistoricalRoundsBufferLen),
		lookupRoundMessages: make(chan roundLookup, params.LookupRoundsBufferLen),
		messageBundles:      bundles,
		sender:              sender,
	}

	m.Internal = internal
	return m
}

func (m *Manager) StartProcessors() stoppable.Stoppable {

	multi := stoppable.NewMulti("Rounds")

	//start the historical rounds thread
	historicalRoundsStopper := stoppable.NewSingle("ProcessHistoricalRounds")
	go m.processHistoricalRounds(m.Comms, historicalRoundsStopper.Quit())
	multi.Add(historicalRoundsStopper)

	//start the message retrieval worker pool
	for i := uint(0); i < m.params.NumMessageRetrievalWorkers; i++ {
		stopper := stoppable.NewSingle(fmt.Sprintf("Messager Retriever %v", i))
		go m.processMessageRetrieval(m.Comms, stopper.Quit())
		multi.Add(stopper)
	}

	// Start the periodic unchecked round worker
	stopper := stoppable.NewSingle("UncheckRound")
	go m.processUncheckedRounds(m.params.UncheckRoundPeriod, newBackoffTable(), stopper.Quit())
	multi.Add(stopper)

	return multi
}
