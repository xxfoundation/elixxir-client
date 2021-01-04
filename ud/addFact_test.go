package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

type testAFC struct{}

// Dummy implementation of SendRegisterFact so we don't need
// to run our own UDB server
func (rFC *testAFC) SendRegisterFact(host *connect.Host, message *pb.FactRegisterRequest) (*pb.FactRegisterResponse, error) {
	return &pb.FactRegisterResponse{}, nil
}

// Test that the addFact function completes successfully
func TestAddFact(t *testing.T) {
	isReg := uint32(1)
	// Add our host, addFact uses it to get the ID of the user
	h, err := connect.NewHost(&id.DummyUser, "address", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}

	// Create a new Private Key to use for signing the Fact
	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	// Create our Manager object
	m := Manager{
		host:       h,
		privKey:    cpk,
		registered: &isReg,
	}

	// Create our test fact
	USCountryCode := "US"
	USNumber := "6502530000"
	f := fact.Fact{
		Fact: USNumber + USCountryCode,
		T:    2,
	}

	// Setup a dummy comms that implements SendRegisterFact
	// This way we don't need to run UDB just to check that this
	// function works.
	tafc := testAFC{}

	// Run addFact and see if it returns without an error!
	_, err = m.addFact(f, &tafc)
	if err != nil {
		t.Fatal(err)
	}
}
