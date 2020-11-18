package ud

import (
"gitlab.com/elixxir/client/interfaces/contact"
"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
"gitlab.com/xx_network/comms/connect"
"gitlab.com/xx_network/crypto/csprng"
"gitlab.com/xx_network/crypto/signature/rsa"
"gitlab.com/xx_network/primitives/id"
"testing"
)

type testAFC struct{}

func (rFC *testAFC) SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error) {
	return &pb.FactRegisterResponse{}, nil
}

func TestAddFact(t *testing.T) {
	c, err := client.NewClientComms(&id.DummyUser, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

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
		comms:   c,
		host:    h,
		privKey: cpk,
	}

	f := contact.Fact{
		Fact: "testing",
		T:    2,
	}

	tafc := testAFC{}

	_, err = m.addFact(f, &tafc)
	if err != nil {
		t.Fatal(err)
	}
}

