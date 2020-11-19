package ud

import (
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

func (rFC *testRFC) SendDeleteMessage(host *connect.Host, message *pb.FactRemovalRequest) (*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func TestRemoveFact(t *testing.T) {
	h, err := connect.NewHost(&id.DummyUser, "address", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}

	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	m := Manager{
		comms:   nil,
		host:    h,
		privKey: cpk,
	}

	f := fact.Fact{
		Fact: "testing",
		T:    2,
	}

	trfc := testRFC{}

	err = m.removeFact(f, &trfc)
	if err != nil {
		t.Fatal(err)
	}
}
