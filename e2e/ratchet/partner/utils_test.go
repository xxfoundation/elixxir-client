package partner

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
)

type mockCyHandler struct {
}

func (m mockCyHandler) AddKey(k *session.Cypher) {
	return
}

func (m mockCyHandler) DeleteKey(k *session.Cypher) {
	return
}

func getGroup() *cyclic.Group {
	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))

	return e2eGrp

}

// newTestManager returns a new relationship for testing.
func newTestManager(t *testing.T) (manager, *versioned.KV) {
	if t == nil {
		panic("Cannot run this outside tests")
	}

	grp := getGroup()
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
		grp, rng.GetStream())
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)
	myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng.GetStream())

	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng.GetStream())
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng.GetStream())
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	kv := versioned.NewKV(make(ekv.Memstore))
	partnerID := id.NewIdFromString("partner", id.User, t)

	myId := id.NewIdFromString("me", id.User, t)

	// Create new relationship
	m := NewManager(kv, myId, partnerID, myPrivKey, partnerPubKey,
		mySIDHPrivKey, partnerSIDHPubKey,
		session.GetDefaultParams(), session.GetDefaultParams(),
		mockCyHandler{}, grp,
		fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG))

	newM := m.(*manager)

	return *newM, kv
}

func managersEqual(expected, received *manager, t *testing.T) bool {
	equal := true
	if !reflect.DeepEqual(expected.cyHandler, received.cyHandler) {
		t.Errorf("Did not Receive expected Manager.cyHandler."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.cyHandler, received.cyHandler)
		equal = false
	}
	if !reflect.DeepEqual(expected.kv, received.kv) {
		t.Errorf("Did not Receive expected Manager.kv."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.kv, received.kv)
		equal = false
	}
	if !expected.partner.Cmp(received.partner) {
		t.Errorf("Did not Receive expected Manager.partner."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.partner, received.partner)
		equal = false
	}
	if !relationshipsEqual(expected.receive, received.receive) {
		t.Errorf("Did not Receive expected Manager.Receive."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.receive, received.receive)
		equal = false
	}
	if !relationshipsEqual(expected.send, received.send) {
		t.Errorf("Did not Receive expected Manager.Send."+
			"\n\texpected: %+v\n\treceived: %+v",
			expected.send, received.send)
		equal = false
	}

	return equal
}

// Compare certain fields of two session buffs for equality
func relationshipsEqual(buff *relationship, buff2 *relationship) bool {
	if len(buff.sessionByID) != len(buff2.sessionByID) {
		return false
	}
	if len(buff.sessions) != len(buff2.sessions) {
		return false
	}

	if !bytes.Equal(buff.fingerprint, buff2.fingerprint) {
		return false
	}
	// Make sure all sessions are present
	for k := range buff.sessionByID {
		_, ok := buff2.sessionByID[k]
		if !ok {
			// key not present in other map
			return false
		}
	}
	// Comparing base key only for now
	// This should ensure that the session buffers have the same sessions in the same order
	for i := range buff.sessions {
		if buff.sessions[i].GetBaseKey().Cmp(buff2.sessions[i].GetBaseKey()) != 0 {
			return false
		}
	}
	return true
}
