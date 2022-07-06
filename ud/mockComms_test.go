package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
)

type mockComms struct {
	udHost *connect.Host
}

func (m mockComms) SendRegisterUser(host *connect.Host, message *pb.UDBUserRegistration) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error) {
	return nil, nil
}

func (m mockComms) SendConfirmFact(host *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRemoveFact(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m mockComms) SendRemoveUser(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	return nil, nil
}

func (m *mockComms) AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	h, err := connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	m.udHost = h
	return h, nil
}

func (m mockComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	return m.udHost, true
}
