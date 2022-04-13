package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
)

// Comms is a sub-interface of the client.Comms interface. This contains
// RPCs relevant to
// todo: docsting on what it is, why it's needed. This is half finished as is
type Comms interface {
	// todo: docsting on what it is, why it's needed
	SendRegisterUser(host *connect.Host, message *pb.UDBUserRegistration) (*messages.Ack, error)
	// todo: docsting on what it is, why it's needed
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
	// todo: docsting on what it is, why it's needed
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
	// todo: docsting on what it is, why it's needed
	SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
	// todo: docsting on what it is, why it's needed
	SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
	// todo: docsting on what it is, why it's needed
	AddHost(hid *id.ID, address string,
		cert []byte, params connect.HostParams) (host *connect.Host, err error)
	// todo: docsting on what it is, why it's needed
	GetHost(hostId *id.ID) (*connect.Host, bool)
}

// todo: docsting on what it is, why it's needed
type removeFactComms interface {
	SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
}

type removeUserComms interface {
	SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
}

type confirmFactComm interface {
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
}

type registerUserComms interface {
	SendRegisterUser(*connect.Host, *pb.UDBUserRegistration) (*messages.Ack, error)
}

type addFactComms interface {
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
}
