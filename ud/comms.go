package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
)

// Comms is a sub-interface of the client.Comms interface. This contains
// RPCs and methods relevant to sending to the UD service.
type Comms interface {
	// SendRegisterUser is the gRPC send function for the user registering
	// their username with the UD service.
	SendRegisterUser(host *connect.Host, message *pb.UDBUserRegistration) (*messages.Ack, error)
	// SendRegisterFact is the gRPC send function for the user registering
	// a fact.Fact (email/phone number) with the UD service.
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
	// SendConfirmFact is the gRPC send function for the user confirming
	// their fact.Fact has been registered successfully with the UD service.
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
	// SendRemoveFact is the gRPC send function for the user removing
	// a registered fact.Fact from the UD service. This fact.Fact must be
	// owned by the user.
	SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
	// SendRemoveUser is the gRPC send function for the user removing
	// their username from the UD service.
	SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
	// AddHost is a function which adds a connect.Host object to the internal
	// comms manager. This will be used here exclusively for adding
	// the UD service if it does not currently exist within the internal
	// manger.
	AddHost(hid *id.ID, address string,
		cert []byte, params connect.HostParams) (host *connect.Host, err error)
	// GetHost retrieves a connect.Host object from the internal comms manager.
	// This will be used exclusively to retrieve the UD service's connect.Host
	// object. This will be used to send to the UD service on the above
	// gRPC send functions.
	GetHost(hostId *id.ID) (*connect.Host, bool)

	SendChannelAuthRequest(host *connect.Host, message *pb.ChannelAuthenticationRequest) (*pb.ChannelAuthenticationResponse, error)
}

// removeFactComms is a sub-interface of the Comms interface for the
// removeFact comm.
type removeFactComms interface {
	SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
}

// removeUserComms is a sub-interface of the Comms interface for the
// permanentDeleteAccount comm.
type removeUserComms interface {
	SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error)
}

// confirmFactComm is a sub-interface of the Comms interface for the
// confirmFact comm.
type confirmFactComm interface {
	SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error)
}

// registerUserComms is a sub-interface of the Comms interface for the
// registerUser comm.
type registerUserComms interface {
	SendRegisterUser(*connect.Host, *pb.UDBUserRegistration) (*messages.Ack, error)
}

// addFactComms is a sub-interface of the Comms interface for the
// addFact comms
type addFactComms interface {
	SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error)
}

type channelLeaseComms interface {
	SendChannelAuthRequest(host *connect.Host, message *pb.ChannelAuthenticationRequest) (*pb.ChannelAuthenticationResponse, error)
}
