package ud

import (
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
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
	storageSess := storage.InitTestingSession(t)

	kv := versioned.NewKV(ekv.Memstore{})
	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("Failed to initialize store %v", err)
	}

	// Create our Manager object
	m := &Manager{
		network: newTestNetworkManager(t),
		e2e:     mockE2e{},
		events:  event.NewEventManager(),
		user:    storageSess,
		comms:   &mockComms{},
		store:   udStore,
		kv:      kv,
		rng:     fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
	}

	c := &testComm{}

	expectedRequest := &pb.FactConfirmRequest{
		ConfirmationID: "test",
		Code:           "1234",
	}

	// Set up store for expected state
	err = m.store.StoreUnconfirmedFact(expectedRequest.ConfirmationID, fact.Fact{})
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
