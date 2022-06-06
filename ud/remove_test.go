package ud

import (
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

type testRFC struct{}

func (rFC *testRFC) SendRemoveFact(*connect.Host, *pb.FactRemovalRequest) (
	*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func TestRemoveFact(t *testing.T) {
	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	isReg := uint32(1)

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Set up manager
	m := &Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		privKey:    cpk,
		registered: &isReg,
		storage:    storage.InitTestingSession(t),
		myID:       &id.ID{},
	}

	f := fact.Fact{
		Fact: "testing",
		T:    2,
	}

	// Set up storage for expected state
	confirmId := "test"
	if err = m.storage.GetUd().StoreUnconfirmedFact(confirmId, f); err != nil {
		t.Fatalf("StoreUnconfirmedFact error: %v", err)
	}

	if err = m.storage.GetUd().ConfirmFact(confirmId); err != nil {
		t.Fatalf("ConfirmFact error: %v", err)
	}

	tRFC := testRFC{}

	err = m.removeFact(f, &tRFC)
	if err != nil {
		t.Fatal(err)
	}
}

func (rFC *testRFC) SendRemoveUser(*connect.Host, *pb.FactRemovalRequest) (
	*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func TestRemoveUser(t *testing.T) {

	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	isReg := uint32(1)

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Set up manager
	m := &Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		privKey:    cpk,
		registered: &isReg,
		myID:       &id.ID{},
	}

	f := fact.Fact{
		Fact: "testing",
		T:    2,
	}

	tRFC := testRFC{}

	err = m.removeUser(f, &tRFC)
	if err != nil {
		t.Fatal(err)
	}
}
