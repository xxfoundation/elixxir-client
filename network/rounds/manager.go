package rounds

import (
	"fmt"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/network/message"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type Manager struct {
	params params.Rounds

	p *processing

	comms    *client.Comms
	instance *network.Instance
	rngGen   *fastRNG.StreamGenerator
	session  *storage.Session

	historicalRounds    chan id.Round
	lookupRoundMessages chan *mixmessages.RoundInfo
	messageBundles      chan message.Bundle
}

func New(comms *client.Comms, instance *network.Instance, session *storage.Session,
	rngGen *fastRNG.StreamGenerator, bundles chan message.Bundle,
	params params.Rounds) (*Manager, error) {
	return &Manager{
		params:   params,
		p:        newProcessingRounds(),
		comms:    comms,
		instance: instance,
		rngGen:   rngGen,
		session:  session,

		historicalRounds:    make(chan id.Round, params.HistoricalRoundsBufferLen),
		lookupRoundMessages: make(chan *mixmessages.RoundInfo, params.LookupRoundsBufferLen),
		messageBundles:      bundles,
	}, nil
}

func (m *Manager) StartProcessors() stoppable.Stoppable {

	multi := stoppable.NewMulti("Rounds")

	//start the historical rounds thread
	historicalRoundsStopper := stoppable.NewSingle("ProcessHistoricalRounds")
	go m.processHistoricalRounds(m.comms, historicalRoundsStopper.Quit())
	multi.Add(historicalRoundsStopper)

	//start the message retrieval worker pool
	for i := uint(0); i < m.params.NumMessageRetrievalWorkers; i++ {
		stopper := stoppable.NewSingle(fmt.Sprintf("Messager Retriever %v", i))
		go m.processMessageRetrieval(m.comms, stopper.Quit())
		multi.Add(stopper)
	}

}
