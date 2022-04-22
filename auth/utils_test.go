package auth

import (
	"gitlab.com/elixxir/client/auth/store"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

type mockSentRequestHandler struct{}

func (msrh *mockSentRequestHandler) Add(sr *store.SentRequest)    {}
func (msrh *mockSentRequestHandler) Delete(sr *store.SentRequest) {}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D4941"+
			"3394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688"+
			"B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861"+
			"575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC"+
			"718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FF"+
			"B1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBC"+
			"A23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD"+
			"161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2B"+
			"EEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C"+
			"4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F"+
			"1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F"+
			"96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA7554715"+
			"84A09EC723742DC35873847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
}

// randID returns a new random ID of the specified type.
func randID(rng *rand.Rand, t id.Type) *id.ID {
	newID, _ := id.NewRandomID(rng, t)
	return newID
}

func newPayload(size int, s string) []byte {
	b := make([]byte, size)
	copy(b[:], s)
	return b
}

func newOwnership(s string) []byte {
	ownership := make([]byte, ownershipSize)
	copy(ownership[:], s)
	return ownership
}

func makeTestRound(t *testing.T) rounds.Round {
	nids := []*id.ID{
		id.NewIdFromString("one", id.User, t),
		id.NewIdFromString("two", id.User, t),
		id.NewIdFromString("three", id.User, t)}
	r := rounds.Round{
		ID:               2,
		State:            states.REALTIME,
		Topology:         connect.NewCircuit(nids),
		Timestamps:       nil,
		Errors:           nil,
		BatchSize:        0,
		AddressSpaceSize: 0,
		UpdateID:         0,
		Raw: &mixmessages.RoundInfo{
			ID:                         5,
			UpdateID:                   0,
			State:                      2,
			BatchSize:                  5,
			Topology:                   [][]byte{[]byte("test"), []byte("test")},
			Timestamps:                 []uint64{uint64(netTime.Now().UnixNano()), uint64(netTime.Now().UnixNano())},
			Errors:                     nil,
			ClientErrors:               nil,
			ResourceQueueTimeoutMillis: 0,
			Signature:                  nil,
			AddressSpaceSize:           0,
			EccSignature:               nil,
		},
	}
	return r
}
