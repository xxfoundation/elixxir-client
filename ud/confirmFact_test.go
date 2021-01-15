package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/primitives/id"
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
	// Create new host
	host, err := connect.NewHost(&id.UDB, "0.0.0.0", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatalf("Could not create a new host: %+v", err)
	}

	isReg := uint32(1)

	// Set up manager
	m := &Manager{
		host: host,
		registered:&isReg,
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
