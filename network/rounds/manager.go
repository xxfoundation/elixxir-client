package rounds

import (
	"fmt"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	params params.Rounds

	p *processing

	internal.Internal

	historicalRounds    chan id.Round
	lookupRoundMessages chan *mixmessages.RoundInfo
	messageBundles      chan<- message.Bundle
}

func NewManager(internal internal.Internal, params params.Rounds,
	bundles chan<- message.Bundle) *Manager {
	m := &Manager{
		params: params,
		p:      newProcessingRounds(),

		historicalRounds:    make(chan id.Round, params.HistoricalRoundsBufferLen),
		lookupRoundMessages: make(chan *mixmessages.RoundInfo, params.LookupRoundsBufferLen),
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
