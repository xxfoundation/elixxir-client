package ud

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

type testAFC struct{}

// Dummy implementation of SendRegisterFact so that we don't need to run our own
// UDB server.
func (rFC *testAFC) SendRegisterFact(*connect.Host, *pb.FactRegisterRequest) (
	*pb.FactRegisterResponse, error) {
	return &pb.FactRegisterResponse{}, nil
}

// Test that the addFact function completes successfully
func TestAddFact(t *testing.T) {
	isReg := uint32(1)

	// Create a new Private Key to use for signing the Fact
	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	comms, err := client.NewClientComms(nil, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}

	// Create our Manager object
	m := Manager{
		comms:      comms,
		net:        newTestNetworkManager(t),
		privKey:    cpk,
		registered: &isReg,
		storage:    storage.InitTestingSession(t),
	}

	// Create our test fact
	USCountryCode := "US"
	USNumber := "6502530000"
	f := fact.Fact{
		Fact: USNumber + USCountryCode,
		T:    2,
	}

	// Set up a dummy comms that implements SendRegisterFact
	// This way we don't need to run UDB just to check that this
	// function works.
	tAFC := testAFC{}
	uid := &id.ID{}
	// Run addFact and see if it returns without an error!
	_, err = m.addFact(f, uid, &tAFC)
	if err != nil {
		t.Fatal(err)
	}
}
