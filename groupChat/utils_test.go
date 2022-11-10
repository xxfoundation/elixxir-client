////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"encoding/base64"
	"gitlab.com/elixxir/client/v5/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v5/event"
	gs "gitlab.com/elixxir/client/v5/groupChat/groupStore"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

/////////////////////////////////////////////////////////////////////////////////////////
// mock manager implementation //////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////////

// newTestManager creates a new manager for testing.
func newTestManager(t testing.TB) (*manager, gs.Group) {
	prng := rand.New(rand.NewSource(42))
	mockMess := newMockE2e(t, nil)

	m := &manager{
		user: mockMess,
	}
	user := group.Member{
		ID:    m.getReceptionIdentity().ID,
		DhKey: m.getE2eHandler().GetHistoricalDHPubkey(),
	}

	g := newTestGroupWithUser(m.getE2eGroup(), user.ID, user.DhKey,
		m.getE2eHandler().GetHistoricalDHPrivkey(), prng, t)
	gStore, err := gs.NewStore(versioned.NewKV(ekv.MakeMemstore()), user)
	if err != nil {
		t.Fatalf("Failed to create new group store: %+v", err)
	}

	m.gs = gStore
	return m, g
}

// newTestManager creates a new manager that has groups stored for testing. One
// of the groups in the list is also returned.
func newTestManagerWithStore(rng *rand.Rand, numGroups int, sendErr int,
	requestFunc RequestCallback, t *testing.T) (*manager, gs.Group) {
	mockMess := newMockE2eWithStore(t, sendErr)

	m := &manager{
		services:    make(map[string]Processor),
		requestFunc: requestFunc,
		user:        mockMess,
	}
	user := group.Member{
		ID:    m.getReceptionIdentity().ID,
		DhKey: m.getE2eHandler().GetHistoricalDHPubkey(),
	}

	gStore, err := gs.NewStore(versioned.NewKV(ekv.MakeMemstore()), user)
	if err != nil {
		t.Fatalf("Failed to create new group store: %+v", err)
	}
	m.gs = gStore

	var g gs.Group
	for i := 0; i < numGroups; i++ {
		g = newTestGroupWithUser(m.getE2eGroup(), user.ID, user.DhKey,
			randCycInt(rng), rng, t)
		if err = gStore.Add(g); err != nil {
			t.Fatalf("Failed to add group %d to group store: %+v", i, err)
		}
	}
	return m, g
}

func newTestE2eManager(dhPubKey *cyclic.Int, t testing.TB) *testE2eManager {
	return &testE2eManager{
		e2eMessages: []testE2eMessage{},
		errSkip:     0,
		grp:         getGroup(),
		dhPubKey:    dhPubKey,
		partners:    make(map[id.ID]partner.Manager),
	}
}

// getMembership returns a Membership with random members for testing.
func getMembership(size int, uid *id.ID, pubKey *cyclic.Int, grp *cyclic.Group,
	prng *rand.Rand, t testing.TB) group.Membership {
	contacts := make([]contact.Contact, size)
	for i := range contacts {
		randId, _ := id.NewRandomID(prng, id.User)
		contacts[i] = contact.Contact{
			ID:       randId,
			DhPubKey: grp.NewInt(int64(prng.Int31() + 1)),
		}
	}

	contacts[2].ID = uid
	contacts[2].DhPubKey = pubKey

	membership, err := group.NewMembership(contacts[0], contacts[1:]...)
	if err != nil {
		t.Errorf("Failed to create new membership: %+v", err)
	}

	return membership
}

// newTestGroup generates a new group with random values for testing.
func newTestGroup(grp *cyclic.Group, privKey *cyclic.Int, rng *rand.Rand,
	t *testing.T) gs.Group {
	// Generate name from base 64 encoded random data
	nameBytes := make([]byte, 16)
	rng.Read(nameBytes)
	name := []byte(base64.StdEncoding.EncodeToString(nameBytes))

	// Generate the message from base 64 encoded random data
	msgBytes := make([]byte, 128)
	rng.Read(msgBytes)
	msg := []byte(base64.StdEncoding.EncodeToString(msgBytes))

	membership := getMembership(10, id.NewIdFromString("userID", id.User, t),
		randCycInt(rng), grp, rng, t)

	dkl := gs.GenerateDhKeyList(
		id.NewIdFromString("userID", id.User, t), privKey, membership, grp)

	idPreimage, err := group.NewIdPreimage(rng)
	if err != nil {
		t.Fatalf("Failed to generate new group ID preimage: %+v", err)
	}

	keyPreimage, err := group.NewKeyPreimage(rng)
	if err != nil {
		t.Fatalf("Failed to generate new group key preimage: %+v", err)
	}

	groupID := group.NewID(idPreimage, membership)
	groupKey := group.NewKey(keyPreimage, membership)

	return gs.NewGroup(name, groupID, groupKey, idPreimage, keyPreimage, msg,
		netTime.Now(), membership, dkl)
}

// newTestGroup generates a new group with random values for testing.
func newTestGroupWithUser(grp *cyclic.Group, uid *id.ID, pubKey,
	privKey *cyclic.Int, rng *rand.Rand, t testing.TB) gs.Group {
	// Generate name from base 64 encoded random data
	nameBytes := make([]byte, 16)
	rng.Read(nameBytes)
	name := []byte(base64.StdEncoding.EncodeToString(nameBytes))

	// Generate the message from base 64 encoded random data
	msgBytes := make([]byte, 128)
	rng.Read(msgBytes)
	msg := []byte(base64.StdEncoding.EncodeToString(msgBytes))

	membership := getMembership(10, uid, pubKey, grp, rng, t)

	dkl := gs.GenerateDhKeyList(uid, privKey, membership, grp)

	idPreimage, err := group.NewIdPreimage(rng)
	if err != nil {
		t.Fatalf("Failed to generate new group ID preimage: %+v", err)
	}

	keyPreimage, err := group.NewKeyPreimage(rng)
	if err != nil {
		t.Fatalf("Failed to generate new group key preimage: %+v", err)
	}

	groupID := group.NewID(idPreimage, membership)
	groupKey := group.NewKey(keyPreimage, membership)

	return gs.NewGroup(name, groupID, groupKey, idPreimage, keyPreimage, msg,
		netTime.Now().Round(0), membership, dkl)
}

// randCycInt returns a random cyclic int.
func randCycInt(rng *rand.Rand) *cyclic.Int {
	return getGroup().NewInt(int64(rng.Int31() + 1))
}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString(getNDF().E2E.Prime, 16),
		large.NewIntFromString(getNDF().E2E.Generator, 16))
}

type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(int, string, string, string) {}
func (tnm *testNetworkManager) GetEventManager() event.Reporter {
	return &dummyEventMgr{}
}

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A" +
				"8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3D" +
				"D2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E78615" +
				"75E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC" +
				"6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C" +
				"4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F2" +
				"6E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE" +
				"448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E" +
				"198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FF" +
				"DDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323" +
				"631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C" +
				"3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E63" +
				"19BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC3" +
				"5873847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642" +
				"F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757" +
				"264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F" +
				"9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091E" +
				"B51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D" +
				"0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D3" +
				"92145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A" +
				"2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7" +
				"995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E2480" +
				"9670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D" +
				"1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A33" +
				"8661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361" +
				"C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B28" +
				"5DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD929" +
				"59859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D83" +
				"2186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8" +
				"B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}
