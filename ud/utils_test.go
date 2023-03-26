////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/event"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	store "gitlab.com/elixxir/client/v4/ud/store"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"io"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

func newTestManager(t *testing.T) (*Manager, *testNetworkManager) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("Failed to initialize store %v", err)
	}

	sch := rsa.GetScheme()

	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	stream := rngGen.GetStream()
	privKey, err := sch.Generate(stream, 1024)
	stream.Close()

	// Create our Manager object
	tnm := newTestNetworkManager(t)
	m := &Manager{
		user: mockE2e{
			grp:       getGroup(),
			events:    event.NewEventManager(),
			mockStore: mockStorage{kv: kv},
			rng:       rngGen,
			kv:        kv,
			network:   tnm,
			t:         t,
			key:       privKey,
		},
		store: udStore,
		comms: &mockComms{},
	}

	netDef := m.getCmix().GetInstance().GetPartialNdf().Get()
	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		t.Fatalf("failed to "+
			"unmarshal UD ID from NDF: %+v", err)
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	params.SendTimeout = 20 * time.Second

	// Add a new host and return it if it does not already exist
	host, err := m.comms.AddHost(udID, netDef.UDB.Address,
		[]byte(netDef.UDB.Cert), params)
	if err != nil {
		t.Fatalf("User Discovery host " +
			"object could not be constructed.")
	}

	udIdData := netDef.UDB.ID
	udId, err := id.Unmarshal(udIdData)
	if err != nil {
		t.Fatalf(err.Error())
	}

	udDhPubKeyData := netDef.UDB.DhPubKey
	udDhPubKey := getGroup().NewInt(1)
	err = udDhPubKey.UnmarshalJSON(udDhPubKeyData)
	if err != nil {
		t.Fatalf(err.Error())
	}

	udContact := contact.Contact{
		ID:       udId,
		DhPubKey: udDhPubKey,
	}
	m.ud = &userDiscovery{
		host:    host,
		contact: udContact,
	}

	tnm.c = udContact

	return m, tnm
}

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }
func newTestNetworkManager(t *testing.T) *testNetworkManager {
	instanceComms := &connect.ProtoComms{
		Manager: connect.NewManagerTesting(t),
	}
	thisInstance, err := network.NewInstanceTesting(instanceComms, getNDF(),
		getNDF(), getGroup(), getGroup(), t)
	if err != nil {
		t.Fatalf("Failed to create new test instance: %v", err)
	}

	tnm := &testNetworkManager{
		instance:    thisInstance,
		testingFace: t,
	}

	return tnm
}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A"+
			"8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3D"+
			"D2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E78615"+
			"75E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC"+
			"6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C"+
			"4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F2"+
			"6E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE"+
			"448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E"+
			"198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FF"+
			"DDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323"+
			"631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C"+
			"3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E63"+
			"19BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC3"+
			"5873847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
}

type mockUser struct {
	testing *testing.T
	key     rsa.PrivateKey
}

func (m mockUser) PortableUserInfo() user.Info {

	return user.Info{
		TransmissionID:        id.NewIdFromString("test", id.User, m.testing),
		TransmissionSalt:      []byte("test"),
		TransmissionRSA:       m.key,
		ReceptionID:           id.NewIdFromString("test", id.User, m.testing),
		ReceptionSalt:         []byte("test"),
		ReceptionRSA:          m.key,
		Precanned:             false,
		RegistrationTimestamp: 0,
		E2eDhPrivateKey:       getGroup().NewInt(5),
		E2eDhPublicKey:        getGroup().NewInt(6),
	}
}

func (m mockUser) GetReceptionRegistrationValidationSignature() []byte {
	return []byte("test")
}

type mockReceiver struct {
	responses []*Contact
	c         mockChannel
	t         *testing.T
}

func newMockReceiver(c mockChannel, response []*Contact, t *testing.T) *mockReceiver {
	return &mockReceiver{
		c:         c,
		t:         t,
		responses: response,
	}
}

func (receiver *mockReceiver) Callback(req *single.Request,
	_ receptionID.EphemeralIdentity, _ []rounds.Round) {
	if req.GetTag() == SearchTag {
		response := &SearchResponse{}
		response.Contacts = receiver.responses

		responsePayload, err := proto.Marshal(response)
		if err != nil {
			receiver.t.Fatalf("Failed to marshal response message: %v", err)
		}

		_, err = req.Respond(responsePayload,
			cmix.GetDefaultCMIXParams(), 100*time.Millisecond)
		if err != nil {
			receiver.t.Fatalf("Respond error: %v", err)
		}
	} else if req.GetTag() == LookupTag {
		response := &LookupResponse{
			PubKey:   receiver.responses[0].PubKey,
			Username: receiver.responses[0].Username,
		}

		responsePayload, err := proto.Marshal(response)
		if err != nil {
			receiver.t.Fatalf("Failed to marshal response message: %v", err)
		}

		_, err = req.Respond(responsePayload,
			cmix.GetDefaultCMIXParams(), 100*time.Millisecond)
		if err != nil {
			receiver.t.Fatalf("Respond error: %v", err)
		}

	}

}

type mockReporter struct{}

func (m mockReporter) Report(priority int, category, evtType, details string) {
	return
}

type mockResponse struct {
	c   []contact.Contact
	err error
}

type mockChannel chan mockResponse

func getNDF() *ndf.NetworkDefinition {

	return &ndf.NetworkDefinition{
		UDB: ndf.UDB{
			ID:       id.DummyUser.Bytes(),
			Cert:     testCert,
			Address:  "address",
			DhPubKey: []byte{123, 34, 86, 97, 108, 117, 101, 34, 58, 53, 48, 49, 53, 53, 53, 52, 54, 53, 49, 48, 54, 49, 56, 57, 53, 54, 51, 48, 54, 52, 49, 51, 53, 49, 57, 56, 55, 57, 52, 57, 50, 48, 56, 49, 52, 57, 52, 50, 57, 51, 57, 53, 49, 50, 51, 54, 52, 56, 49, 57, 55, 48, 50, 50, 49, 48, 55, 55, 50, 52, 52, 48, 49, 54, 57, 52, 55, 52, 57, 53, 53, 56, 55, 54, 50, 57, 53, 57, 53, 48, 54, 55, 57, 55, 48, 53, 48, 48, 54, 54, 56, 49, 57, 50, 56, 48, 52, 48, 53, 51, 50, 48, 57, 55, 54, 56, 56, 53, 57, 54, 57, 56, 57, 49, 48, 54, 56, 54, 50, 52, 50, 52, 50, 56, 49, 48, 51, 51, 51, 54, 55, 53, 55, 54, 52, 51, 54, 55, 54, 56, 53, 56, 48, 55, 56, 49, 52, 55, 49, 52, 53, 49, 52, 52, 52, 52, 53, 51, 57, 57, 51, 57, 57, 53, 50, 52, 52, 53, 51, 56, 48, 49, 48, 54, 54, 55, 48, 52, 50, 49, 55, 54, 57, 53, 57, 57, 57, 51, 52, 48, 54, 54, 54, 49, 50, 48, 54, 56, 57, 51, 54, 57, 48, 52, 55, 55, 54, 50, 49, 49, 56, 56, 53, 51, 50, 57, 57, 50, 54, 53, 48, 52, 57, 51, 54, 55, 54, 48, 57, 56, 56, 49, 55, 52, 52, 57, 53, 57, 54, 53, 50, 55, 53, 52, 52, 52, 49, 57, 55, 49, 54, 50, 52, 52, 56, 50, 55, 55, 50, 49, 48, 53, 56, 56, 57, 54, 51, 53, 54, 54, 53, 53, 53, 53, 49, 56, 50, 53, 49, 49, 50, 57, 50, 48, 49, 56, 48, 48, 54, 49, 56, 57, 48, 55, 48, 51, 53, 51, 51, 56, 57, 52, 49, 50, 57, 49, 55, 50, 56, 55, 57, 57, 52, 55, 53, 51, 49, 55, 55, 48, 53, 55, 55, 49, 50, 51, 57, 49, 51, 55, 54, 48, 50, 49, 55, 50, 54, 54, 52, 56, 52, 48, 48, 54, 48, 52, 48, 53, 56, 56, 53, 54, 52, 56, 56, 49, 52, 52, 51, 57, 56, 51, 51, 57, 54, 55, 48, 49, 53, 55, 52, 53, 50, 56, 51, 49, 51, 48, 53, 52, 49, 49, 49, 49, 49, 56, 51, 53, 52, 52, 52, 52, 48, 53, 54, 57, 48, 54, 52, 56, 57, 52, 54, 53, 50, 56, 51, 53, 50, 48, 48, 50, 48, 48, 49, 50, 51, 51, 48, 48, 53, 48, 49, 50, 52, 56, 57, 48, 49, 51, 54, 55, 52, 57, 55, 50, 49, 48, 55, 53, 54, 49, 50, 52, 52, 57, 55, 48, 50, 56, 55, 55, 51, 51, 50, 53, 50, 48, 57, 52, 56, 57, 49, 49, 56, 49, 54, 57, 50, 55, 50, 51, 57, 51, 57, 54, 50, 56, 48, 54, 54, 49, 57, 55, 48, 50, 48, 57, 49, 51, 54, 50, 49, 50, 53, 50, 54, 50, 53, 53, 55, 57, 54, 51, 56, 49, 57, 48, 51, 49, 54, 54, 53, 51, 56, 56, 49, 48, 56, 48, 51, 57, 53, 49, 53, 53, 55, 49, 53, 57, 48, 57, 57, 55, 49, 56, 53, 55, 54, 48, 50, 54, 48, 49, 55, 57, 52, 55, 53, 51, 57, 49, 51, 53, 52, 49, 48, 50, 49, 55, 52, 51, 57, 48, 50, 56, 48, 50, 51, 53, 51, 54, 56, 49, 56, 50, 49, 55, 50, 57, 52, 51, 49, 56, 48, 56, 56, 50, 51, 53, 52, 56, 55, 49, 52, 55, 53, 50, 56, 48, 57, 55, 49, 53, 48, 48, 51, 50, 48, 57, 50, 50, 53, 50, 56, 51, 57, 55, 57, 49, 57, 50, 53, 56, 51, 55, 48, 51, 57, 54, 48, 50, 55, 54, 48, 54, 57, 55, 52, 53, 54, 52, 51, 56, 52, 53, 54, 48, 51, 57, 55, 55, 55, 49, 53, 57, 57, 49, 57, 52, 57, 56, 56, 54, 56, 50, 49, 49, 54, 56, 55, 56, 55, 51, 51, 57, 52, 53, 49, 52, 52, 55, 57, 53, 57, 49, 57, 52, 48, 51, 53, 49, 49, 49, 51, 48, 53, 54, 54, 50, 49, 56, 57, 52, 55, 50, 49, 54, 53, 57, 53, 50, 57, 50, 48, 51, 51, 52, 48, 56, 55, 54, 50, 49, 49, 49, 56, 53, 54, 57, 51, 57, 50, 53, 48, 53, 56, 56, 55, 56, 53, 54, 55, 51, 56, 55, 50, 53, 57, 56, 52, 54, 53, 49, 51, 50, 54, 51, 50, 48, 56, 56, 57, 52, 53, 57, 53, 56, 57, 57, 54, 52, 55, 55, 50, 57, 51, 51, 52, 55, 51, 48, 52, 56, 56, 50, 51, 50, 52, 53, 48, 51, 50, 56, 56, 50, 49, 55, 51, 51, 53, 54, 55, 51, 50, 51, 52, 56, 53, 52, 55, 48, 51, 56, 50, 51, 49, 53, 55, 52, 53, 53, 48, 55, 55, 56, 55, 48, 50, 51, 52, 50, 53, 52, 51, 48, 57, 56, 56, 54, 56, 54, 49, 57, 54, 48, 55, 55, 52, 57, 55, 56, 51, 48, 51, 57, 49, 55, 52, 49, 51, 49, 54, 57, 54, 50, 49, 52, 50, 55, 57, 55, 56, 56, 51, 49, 55, 51, 50, 54, 56, 49, 56, 53, 57, 48, 49, 49, 53, 48, 52, 53, 51, 51, 56, 52, 57, 57, 55, 54, 51, 55, 55, 48, 55, 49, 52, 50, 49, 54, 48, 49, 54, 52, 49, 57, 53, 56, 49, 54, 50, 55, 49, 52, 49, 52, 56, 49, 51, 52, 50, 53, 56, 55, 53, 57, 55, 52, 49, 57, 49, 55, 51, 55, 49, 51, 57, 54, 51, 49, 51, 49, 56, 53, 50, 49, 53, 52, 49, 51, 44, 34, 70, 105, 110, 103, 101, 114, 112, 114, 105, 110, 116, 34, 58, 49, 54, 56, 48, 49, 53, 52, 49, 53, 49, 49, 50, 51, 51, 48, 57, 56, 51, 54, 51, 125},
		},
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
