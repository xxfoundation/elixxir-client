package ud

import (
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"reflect"
	"testing"
)

type testComm struct {
	request *pb.FactConfirmRequest
}

func (t *testComm) SendConfirmFact(_ *connect.Host, message *pb.FactConfirmRequest) (*messages.Ack, error) {
	t.request = message
	return &messages.Ack{}, nil
}

// Happy path.
func TestManager_confirmFact(t *testing.T) {
	isReg := uint32(1)

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Set up manager
	m := &Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		registered: &isReg,
	}

	c := &testComm{}

	expectedRequest := &pb.FactConfirmRequest{
		ConfirmationID: "test",
		Code:           "1234",
	}

	err = m.confirmFact(expectedRequest.ConfirmationID, expectedRequest.Code, c)
	if err != nil {
		t.Errorf("confirmFact() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedRequest, c.request) {
		t.Errorf("end point did not recieve the expected request."+
			"\n\texpected: %+v\n\treceived: %+v", expectedRequest, c.request)
	}

}
