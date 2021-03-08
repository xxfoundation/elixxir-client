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
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

type Manager struct {
	params params.Rounds

	p *processing

	internal.Internal

	historicalRounds    chan historicalRoundRequest
	lookupRoundMessages chan roundLookup
	messageBundles      chan<- message.Bundle
}

func NewManager(internal internal.Internal, params params.Rounds,
	bundles chan<- message.Bundle) *Manager {
	m := &Manager{
		params: params,
		p:      newProcessingRounds(),

		historicalRounds:    make(chan historicalRoundRequest, params.HistoricalRoundsBufferLen),
		lookupRoundMessages: make(chan roundLookup, params.LookupRoundsBufferLen),
		messageBundles:      bundles,
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
	return multi
}

func (m *Manager) DeleteProcessingRoundDelete(round id.Round, eph ephemeral.Id, source *id.ID) {

	m.p.Delete(round, eph, source)
}
