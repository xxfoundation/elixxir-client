package message

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	ds "gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/comms/testutils"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
	"time"
)

type MockSendCMIXComms struct {
	t *testing.T
}

func (mc *MockSendCMIXComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	nid1 := id.NewIdFromString("zezima", id.Node, mc.t)
	gwid := nid1.DeepCopy()
	gwid.SetType(id.Gateway)
	h, _ := connect.NewHost(gwid, "0.0.0.0", []byte(""), connect.HostParams{
		MaxRetries:  0,
		AuthEnabled: false,
	})
	return h, true
}

func (mc *MockSendCMIXComms) AddHost(hid *id.ID, address string, cert []byte, params connect.HostParams) (host *connect.Host, err error) {
	host, _ = mc.GetHost(nil)
	return host, nil
}

func (mc *MockSendCMIXComms) RemoveHost(hid *id.ID) {

}

func (mc *MockSendCMIXComms) SendPutMessage(host *connect.Host, message *mixmessages.GatewaySlot) (*mixmessages.GatewaySlotResponse, error) {
	return &mixmessages.GatewaySlotResponse{
		Accepted: true,
		RoundID:  3,
	}, nil
}

func Test_attemptSendCmix(t *testing.T) {
	sess1 := storage.InitTestingSession(t)

	sess2 := storage.InitTestingSession(t)

	sw := switchboard.New()
	l := TestListener{
		ch: make(chan bool),
	}
	sw.RegisterListener(sess2.GetUser().TransmissionID, message.Raw, l)
	comms, err := client.NewClientComms(sess1.GetUser().TransmissionID, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}
	inst, err := network.NewInstanceTesting(comms.ProtoComms, getNDF(), nil, nil, nil, t)
	if err != nil {
		t.Errorf("Failed to start instance: %+v", err)
	}
	now := netTime.Now()
	nid1 := id.NewIdFromString("zezima", id.Node, t)
	nid2 := id.NewIdFromString("jakexx360", id.Node, t)
	nid3 := id.NewIdFromString("westparkhome", id.Node, t)
	grp := cyclic.NewGroup(large.NewInt(7), large.NewInt(13))
	sess1.Cmix().Add(nid1, grp.NewInt(1))
	sess1.Cmix().Add(nid2, grp.NewInt(2))
	sess1.Cmix().Add(nid3, grp.NewInt(3))

	timestamps := []uint64{
		uint64(now.Add(-30 * time.Second).UnixNano()), //PENDING
		uint64(now.Add(-25 * time.Second).UnixNano()), //PRECOMPUTING
		uint64(now.Add(-5 * time.Second).UnixNano()),  //STANDBY
		uint64(now.Add(5 * time.Second).UnixNano()),   //QUEUED
		0} //REALTIME

	ri := &mixmessages.RoundInfo{
		ID:                         3,
		UpdateID:                   0,
		State:                      uint32(states.QUEUED),
		BatchSize:                  0,
		Topology:                   [][]byte{nid1.Marshal(), nid2.Marshal(), nid3.Marshal()},
		Timestamps:                 timestamps,
		Errors:                     nil,
		ClientErrors:               nil,
		ResourceQueueTimeoutMillis: 0,
		Signature:                  nil,
		AddressSpaceSize:           4,
	}

	if err = testutils.SignRoundInfo(ri, t); err != nil {
		t.Errorf("Failed to sign mock round info: %v", err)
	}

	pubKey, err := testutils.LoadPublicKeyTesting(t)
	if err != nil {
		t.Errorf("Failed to load a key for testing: %v", err)
	}
	rnd := ds.NewRound(ri, pubKey)
	inst.GetWaitingRounds().Insert(rnd)
	i := internal.Internal{
		Session:          sess1,
		Switchboard:      sw,
		Rng:              fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		Comms:            comms,
		Health:           nil,
		TransmissionID:   sess1.GetUser().TransmissionID,
		Instance:         inst,
		NodeRegistration: nil,
	}
	p := gateway.DefaultPoolParams()
	p.PoolSize = 1
	sender, err := gateway.NewSender(p, i.Rng.GetStream(), getNDF(), &MockSendCMIXComms{t: t}, i.Session, nil)
	if err != nil {
		t.Errorf("%+v", errors.New(err.Error()))
		return
	}
	m := NewManager(i, params.Messages{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksGarbledMessage:        20,
		GarbledMessageWait:             time.Hour,
	}, nil, sender)
	msgCmix := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
	msgCmix.SetContents([]byte("test"))
	e2e.SetUnencrypted(msgCmix, m.Session.User().GetCryptographicIdentity().GetTransmissionID())
	_, _, err = sendCmixHelper(sender, msgCmix, sess2.GetUser().ReceptionID, params.GetDefaultCMIX(),
		m.Instance, m.Session, m.nodeRegistration, m.Rng,
		m.TransmissionID, &MockSendCMIXComms{t: t})
	if err != nil {
		t.Errorf("Failed to sendcmix: %+v", err)
		panic("t")
		return
	}
}

func getNDF() *ndf.NetworkDefinition {
	nodeId := id.NewIdFromString("zezima", id.Node, &testing.T{})
	gwId := nodeId.DeepCopy()
	gwId.SetType(id.Gateway)
	return &ndf.NetworkDefinition{
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
			},
		},
	}
}
