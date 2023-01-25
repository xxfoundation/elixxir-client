package pickup

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
)

type mockCertCheckerComm struct {
}

func (mccc *mockCertCheckerComm) GetGatewayTLSCertificate(host *connect.Host,
	message *pb.RequestGatewayCert) (*pb.GatewayCertificate, error) {
	return &pb.GatewayCertificate{}, nil
}
