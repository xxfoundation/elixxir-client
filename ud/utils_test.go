////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
	"time"
)

func newTestNetworkManager(t *testing.T) interfaces.NetworkManager {
	instanceComms := &connect.ProtoComms{
		Manager: connect.NewManagerTesting(t),
	}

	thisInstance, err := network.NewInstanceTesting(instanceComms, getNDF(),
		getNDF(), nil, nil, t)
	if err != nil {
		t.Fatalf("Failed to create new test instance: %v", err)
	}

	return &testNetworkManager{
		instance: thisInstance,
	}
}

// testNetworkManager is a test implementation of NetworkManager interface.
type testNetworkManager struct {
	instance *network.Instance
}

func (tnm *testNetworkManager) SendE2E(message.Send, params.E2E, *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error) {
	return nil, e2e.MessageID{}, time.Time{}, nil
}

func (tnm *testNetworkManager) SendUnsafe(message.Send, params.Unsafe) ([]id.Round, error) {
	return nil, nil
}

func (tnm *testNetworkManager) GetVerboseRounds() string {
	return ""
}

func (tnm *testNetworkManager) SendCMIX(format.Message, *id.ID, params.CMIX) (id.Round, ephemeral.Id, error) {
	return 0, ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) SendManyCMIX([]message.TargetedCmixMessage, params.CMIX) (id.Round, []ephemeral.Id, error) {
	return 0, nil, nil
}

type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(int, string, string, string) {}
func (tnm *testNetworkManager) GetEventManager() event.Manager {
	return &dummyEventMgr{}
}

func (tnm *testNetworkManager) GetInstance() *network.Instance             { return tnm.instance }
func (tnm *testNetworkManager) GetHealthTracker() interfaces.HealthTracker { return nil }
func (tnm *testNetworkManager) Follow(interfaces.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}
func (tnm *testNetworkManager) CheckGarbledMessages()        {}
func (tnm *testNetworkManager) InProgressRegistrations() int { return 0 }
func (tnm *testNetworkManager) GetSender() *gateway.Sender   { return nil }
func (tnm *testNetworkManager) GetAddressSize() uint8        { return 0 }
func (tnm *testNetworkManager) RegisterAddressSizeNotification(string) (chan uint8, error) {
	return nil, nil
}
func (tnm *testNetworkManager) UnregisterAddressSizeNotification(string) {}
func (tnm *testNetworkManager) SetPoolFilter(gateway.Filter)             {}

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		UDB: ndf.UDB{
			ID:      id.DummyUser.Bytes(),
			Cert:    "",
			Address: "address",
			DhPubKey: []byte{123, 34, 86, 97, 108, 117, 101, 34, 58, 49, 44, 34,
				70, 105, 110, 103, 101, 114, 112, 114, 105, 110, 116, 34, 58,
				51, 49, 54, 49, 50, 55, 48, 53, 56, 49, 51, 52, 50, 49, 54, 54,
				57, 52, 55, 125},
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
