package message

import (
	"fmt"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message/parse"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
)

type Manager struct {
	param       params.Messages
	partitioner parse.Partitioner
	internal.Internal

	messageReception chan Bundle
	nodeRegistration chan network.NodeGateway
	networkIsHealthy chan bool
	triggerGarbled   chan struct{}
}

func NewManager(internal internal.Internal, param params.Messages,
	nodeRegistration chan network.NodeGateway) *Manager {
	dummyMessage := format.NewMessage(internal.Session.Cmix().GetGroup().GetP().ByteLen())
	m := Manager{
		param:            param,
		partitioner:      parse.NewPartitioner(dummyMessage.ContentsSize(), internal.Session),
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		networkIsHealthy: make(chan bool, 1),
		triggerGarbled:   make(chan struct{}, 1),
		nodeRegistration: nodeRegistration,
	}
	m.Internal = internal
	return &m
}

//Gets the channel to send received messages on
func (m *Manager) GetMessageReceptionChannel() chan<- Bundle {
	return m.messageReception
}

//Gets the channel to send received messages on
func (m *Manager) GetTriggerGarbledCheckChannel() chan<- struct{} {
	return m.triggerGarbled
}

//Starts all worker pool
func (m *Manager) StartProcessies() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	//create the message reception workers
	for i := uint(0); i < m.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("MessageReception Worker %v", i))
		go m.processMessages(stop.Quit())
		multi.Add(stop)
	}

	//create the critical messages thread
	critStop := stoppable.NewSingle("CriticalMessages")
	go m.processCriticalMessages(critStop.Quit())
	m.Health.AddChannel(m.networkIsHealthy)
	multi.Add(critStop)

	//create the garbled messages thread
	garbledStop := stoppable.NewSingle("GarbledMessages")
	go m.processGarbledMessages(garbledStop.Quit())
	multi.Add(garbledStop)


	return multi
}
