////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
)

// Creates new registrar for testing.
func makeTestRegistrar(t *testing.T) (*registrar, *versioned.KV) {
	connect.TestingOnlyDisableTLS = true
	kv := versioned.NewKV(make(ekv.Memstore))
	session := storage.InitTestingSession(t)
	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(gateway.DefaultPoolParams(), rngGen,
		getNDF(), newMockManager(), session, nil)
	if err != nil {
		t.Fatalf("Failed to create new sender: %+v", err)
	}

	r, err := LoadRegistrar(kv, session, sender, NewMockClientComms(), rngGen)
	if err != nil {
		t.Fatalf("Failed to create new registrar: %+v", err)
	}

	return r.(*registrar), kv
}

// Mock client comms object adhering to RegisterNodeCommsInterface for testing.
type MockClientComms struct {
	request chan bool
	confirm chan bool
}

// Constructor for NewMockClientComms.
func NewMockClientComms() *MockClientComms {
	return &MockClientComms{
		request: make(chan bool, 1),
		confirm: make(chan bool, 1),
	}
}

func (mcc *MockClientComms) GetHost(_ *id.ID) (*connect.Host, bool) {
	return &connect.Host{}, true
}
func (mcc *MockClientComms) SendRequestClientKeyMessage(*connect.Host,
	*pb.SignedClientKeyRequest) (*pb.SignedKeyResponse, error) {
	// Use this channel to confirm that request nonce was called
	mcc.request <- true
	return &pb.SignedKeyResponse{
		KeyResponse:      []byte("nonce"),
		ClientGatewayKey: []byte("dhPub"),
	}, nil
}

// Mock structure adhering to gateway.HostManager for testing.
type mockHostManager struct {
	hosts map[string]*connect.Host
}

// Constructor for mockHostManager.
func newMockManager() *mockHostManager {
	return &mockHostManager{make(map[string]*connect.Host)}
}

func (mhp *mockHostManager) GetHost(hostId *id.ID) (*connect.Host, bool) {
	h, ok := mhp.hosts[hostId.String()]
	return h, ok
}

func (mhp *mockHostManager) AddHost(hid *id.ID, address string, cert []byte,
	params connect.HostParams) (host *connect.Host, err error) {
	host, err = connect.NewHost(hid, address, cert, params)
	if err != nil {
		return nil, err
	}

	mhp.hosts[hid.String()] = host

	return
}

func (mhp *mockHostManager) RemoveHost(hid *id.ID) {
	delete(mhp.hosts, hid.String())
}

func getNDF() *ndf.NetworkDefinition {
	nodeId := id.NewIdFromString("zezima", id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
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
		Gateways: []ndf.Gateway{
			{
				ID:             gwId.Marshal(),
				Address:        "0.0.0.0",
				TlsCertificate: "",
			},
		},
		Nodes: []ndf.Node{
			{
				ID:             nodeId.Marshal(),
				Address:        "0.0.0.0",
				TlsCertificate: "",
				Status:         ndf.Active,
			},
		},
	}
}
