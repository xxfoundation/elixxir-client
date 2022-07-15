package groupChat

import (
	"gitlab.com/elixxir/client/cmix"
	clientE2E "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// mockMessenger implementation for messenger interface
type mockMessenger struct {
	receptionId *id.ID
	net         cmix.Client
	e2e         clientE2E.Handler
	e2eGroup    *cyclic.Group
	rng         *fastRNG.StreamGenerator
	storage     storage.Session
}

func newMockMessenger(t testing.TB, kv *versioned.KV) messenger {
	receptionId := id.NewIdFromString("test", id.User, t)
	mockCmix := newTestNetworkManager(0)
	prng := rand.New(rand.NewSource(42))
	e2eHandler := newTestE2eManager(randCycInt(prng), t)
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mockSession := newMockSesion(kv)

	return mockMessenger{
		receptionId: receptionId,
		net:         mockCmix,
		e2e:         e2eHandler,
		e2eGroup:    grp,
		rng:         rng,
		storage:     mockSession,
	}
}

func newMockMessengerWithStore(t testing.TB, sendErr int) messenger {
	receptionId := id.NewIdFromString("test", id.User, t)
	mockCmix := newTestNetworkManager(sendErr)
	prng := rand.New(rand.NewSource(42))
	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mockSession := newMockSesion(nil)

	return mockMessenger{
		receptionId: receptionId,
		net:         mockCmix,
		e2e: &testE2eManager{
			e2eMessages: []testE2eMessage{},
			sendErr:     sendErr,
			grp:         getGroup(),
			dhPubKey:    randCycInt(prng),
			partners:    make(map[id.ID]partner.Manager),
		},
		e2eGroup: grp,
		rng:      rng,
		storage:  mockSession,
	}
}

func (m mockMessenger) GetCmix() cmix.Client {
	return m.net
}

func (m mockMessenger) GetE2E() clientE2E.Handler {
	return m.e2e
}

func (m mockMessenger) GetReceptionIdentity() xxdk.ReceptionIdentity {
	keyData, _ := m.e2e.GetHistoricalDHPrivkey().MarshalJSON()
	groupData, _ := getGroup().MarshalJSON()
	return xxdk.ReceptionIdentity{
		ID:           m.receptionId,
		DHKeyPrivate: keyData,
		E2eGrp:       groupData,
	}
}

func (m mockMessenger) GetRng() *fastRNG.StreamGenerator {
	return m.rng
}

func (m mockMessenger) GetStorage() storage.Session {
	return m.storage
}
