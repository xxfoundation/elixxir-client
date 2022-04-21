package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
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

	// Create our Manager object
	m, _ := newTestManager(t)

	c := &testComm{}

	expectedRequest := &pb.FactConfirmRequest{
		ConfirmationID: "test",
		Code:           "1234",
	}

	// Set up store for expected state
	err := m.store.StoreUnconfirmedFact(expectedRequest.ConfirmationID, fact.Fact{})
	if err != nil {
		t.Fatalf("StoreUnconfirmedFact error: %v", err)
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
