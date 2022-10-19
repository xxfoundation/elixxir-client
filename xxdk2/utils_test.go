////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk2

import (
	"testing"

	"gitlab.com/elixxir/client/cmix/gateway"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
)

func newTestingClient(face interface{}) (*Cmix, error) {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("InitTestingSession is restricted to testing "+
			"only. Got %T", face)
	}

	def := getNDF(face)
	marshalledDef, _ := def.Marshal()
	storageDir := "ignore.1"
	password := []byte("hunter2")
	err := NewCmix(string(marshalledDef), storageDir, password, "AAAA")
	if err != nil {
		return nil, errors.Errorf(
			"Could not construct a mock client: %v", err)
	}

	c, err := OpenCmix(storageDir, password)
	if err != nil {
		return nil, errors.Errorf("Could not open a mock client: %v",
			err)
	}

	commsManager := connect.NewManagerTesting(face)

	cert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		jww.FATAL.Panicf("Failed to create new test instance: %v", err)
	}

	_, err = commsManager.AddHost(
		&id.Permissioning, "", cert, connect.GetDefaultHostParams())
	if err != nil {
		return nil, err
	}
	instanceComms := &connect.ProtoComms{
		Manager: commsManager,
	}

	thisInstance, err := network.NewInstanceTesting(instanceComms, def,
		def, nil, nil, face)
	if err != nil {
		return nil, err
	}

	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(p, c.GetRng(), def, commsManager,
		c.storage, nil)
	if err != nil {
		return nil, err
	}
	c.network = &testNetworkManagerGeneric{instance: thisInstance, sender: sender}

	return c, nil
}

// Helper function that generates an NDF for testing.
func getNDF(face interface{}) *ndf.NetworkDefinition {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("InitTestingSession is restricted to testing only. Got %T", face)
	}

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeID := id.NewIdFromBytes([]byte("gateway"), face)
	gwId := nodeID.DeepCopy()
	gwId.SetType(id.Gateway)
	return &ndf.NetworkDefinition{
		Registration: ndf.Registration{
			TlsCertificate: string(cert),
			EllipticPubKey: "/WRtT+mDZGC3FXQbvuQgfqOonAjJ47IKE0zhaGTQQ70=",
		},
		Nodes: []ndf.Node{
			{
				ID:             nodeID.Bytes(),
				Address:        "",
				TlsCertificate: string(cert),
				Status:         ndf.Active,
			},
		},
		Gateways: []ndf.Gateway{
			{
				ID:             gwId.Bytes(),
				Address:        "",
				TlsCertificate: string(cert),
			},
		},
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
				"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
				"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
				"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
				"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
				"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
				"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
				"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
				"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
				"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
				"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
				"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
				"847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}

// signRoundInfo signs a passed round info with the key tied to the test node's
// cert used throughout utils and other tests.
func signRoundInfo(ri *pb.RoundInfo) error {
	privKeyFromFile := testkeys.LoadFromPath(testkeys.GetNodeKeyPath())

	pk, err := tls.LoadRSAPrivateKey(string(privKeyFromFile))
	if err != nil {
		return errors.Errorf("Couldn't load private key: %+v", err)
	}

	ourPrivateKey := &rsa.PrivateKey{PrivateKey: *pk}

	return signature.SignRsa(ri, ourPrivateKey)

}
