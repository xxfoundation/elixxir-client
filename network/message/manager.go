package message

import (
	"fmt"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/network/message/parse"
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
}

func NewManager(internal internal.Internal, param params.Messages,
	nodeRegistration chan network.NodeGateway) *Manager {
	dummyMessage := format.NewMessage(internal.Session.Cmix().GetGroup().GetP().ByteLen())
	m := Manager{
		param:            param,
		partitioner:      parse.NewPartitioner(dummyMessage.ContentsSize(), internal.Session),
		messageReception: make(chan Bundle, param.MessageReceptionBuffLen),
		networkIsHealthy: make(chan bool, 1),
		nodeRegistration: nodeRegistration,
	}
	m.Internal = internal
	return &m
}

//Gets the channel to send received messages on
func (m *Manager) GetMessageReceptionChannel() chan<- Bundle {
	return m.messageReception
}

//Starts all worker pool
func (m *Manager) StartProcessies() stoppable.Stoppable {
	multi := stoppable.NewMulti("MessageReception")

	for i := uint(0); i < m.param.MessageReceptionWorkerPoolSize; i++ {
		stop := stoppable.NewSingle(fmt.Sprintf("MessageReception Worker %v", i))
		go m.processMessages(stop.Quit())
		multi.Add(stop)
	}

	critStop := stoppable.NewSingle("Critical Messages Handler")
	go m.processCriticalMessages(critStop.Quit())
	m.Health.AddChannel(m.networkIsHealthy)

	return multi
}
