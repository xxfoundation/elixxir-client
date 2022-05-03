package streaming

import (
	"bytes"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/crypto/cyclic"
	ce2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/region"
	"testing"
	"time"
)

var pub = "-----BEGIN CERTIFICATE-----\nMIIGHTCCBAWgAwIBAgIUOcAn9cpH+hyRH8/UfqtbFDoSxYswDQYJKoZIhvcNAQEL\nBQAwgZIxCzAJBgNVBAYTAlVTMQswCQYDVQQIDAJDQTESMBAGA1UEBwwJQ2xhcmVt\nb250MRAwDgYDVQQKDAdFbGl4eGlyMRQwEgYDVQQLDAtEZXZlbG9wbWVudDEZMBcG\nA1UEAwwQZ2F0ZXdheS5jbWl4LnJpcDEfMB0GCSqGSIb3DQEJARYQYWRtaW5AZWxp\neHhpci5pbzAeFw0xOTA4MTYwMDQ4MTNaFw0yMDA4MTUwMDQ4MTNaMIGSMQswCQYD\nVQQGEwJVUzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCUNsYXJlbW9udDEQMA4GA1UE\nCgwHRWxpeHhpcjEUMBIGA1UECwwLRGV2ZWxvcG1lbnQxGTAXBgNVBAMMEGdhdGV3\nYXkuY21peC5yaXAxHzAdBgkqhkiG9w0BCQEWEGFkbWluQGVsaXh4aXIuaW8wggIi\nMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC7Dkb6VXFn4cdpU0xh6ji0nTDQ\nUyT9DSNW9I3jVwBrWfqMc4ymJuonMZbuqK+cY2l+suS2eugevWZrtzujFPBRFp9O\n14Jl3fFLfvtjZvkrKbUMHDHFehascwzrp3tXNryiRMmCNQV55TfITVCv8CLE0t1i\nbiyOGM9ZWYB2OjXt59j76lPARYww5qwC46vS6+3Cn2Yt9zkcrGeskWEFa2VttHqF\n910TP+DZk2R5C7koAh6wZYK6NQ4S83YQurdHAT51LKGrbGehFKXq6/OAXCU1JLi3\nkW2PovTb6MZuvxEiRmVAONsOcXKu7zWCmFjuZZwfRt2RhnpcSgzfrarmsGM0LZh6\nJY3MGJ9YdPcVGSz+Vs2E4zWbNW+ZQoqlcGeMKgsIiQ670g0xSjYICqldpt79gaET\n9PZsoXKEmKUaj6pq1d4qXDk7s63HRQazwVLGBdJQK8qX41eCdR8VMKbrCaOkzD5z\ngnEu0jBBAwdMtcigkMIk1GRv91j7HmqwryOBHryLi6NWBY3tjb4So9AppDQB41SH\n3SwNenAbNO1CXeUqN0hHX6I1bE7OlbjqI7tXdrTllHAJTyVVjenPel2ApMXp+LVR\ndDbKtwBiuM6+n+z0I7YYerxN1gfvpYgcXm4uye8dfwotZj6H2J/uSALsU2v9UHBz\nprdrLSZk2YpozJb+CQIDAQABo2kwZzAdBgNVHQ4EFgQUDaTvG7SwgRQ3wcYx4l+W\nMcZjX7owHwYDVR0jBBgwFoAUDaTvG7SwgRQ3wcYx4l+WMcZjX7owDwYDVR0TAQH/\nBAUwAwEB/zAUBgNVHREEDTALgglmb28uY28udWswDQYJKoZIhvcNAQELBQADggIB\nADKz0ST0uS57oC4rT9zWhFqVZkEGh1x1XJ28bYtNUhozS8GmnttV9SnJpq0EBCm/\nr6Ub6+Wmf60b85vCN5WDYdoZqGJEBjGGsFzl4jkYEE1eeMfF17xlNUSdt1qLCE8h\nU0glr32uX4a6nsEkvw1vo1Liuyt+y0cOU/w4lgWwCqyweu3VuwjZqDoD+3DShVzX\n8f1p7nfnXKitrVJt9/uE+AtAk2kDnjBFbRxCfO49EX4Cc5rADUVXMXm0itquGBYp\nMbzSgFmsMp40jREfLYRRzijSZj8tw14c2U9z0svvK9vrLCrx9+CZQt7cONGHpr/C\n/GIrP/qvlg0DoLAtjea73WxjSCbdL3Nc0uNX/ymXVHdQ5husMCZbczc9LYdoT2VP\nD+GhkAuZV9g09COtRX4VP09zRdXiiBvweiq3K78ML7fISsY7kmc8KgVH22vcXvMX\nCgGwbrxi6QbQ80rWjGOzW5OxNFvjhvJ3vlbOT6r9cKZGIPY8IdN/zIyQxHiim0Jz\noavr9CPDdQefu9onizsmjsXFridjG/ctsJxcUEqK7R12zvaTxu/CVYZbYEUFjsCe\nq6ZAACiEJGvGeKbb/mSPvGs2P1kS70/cGp+P5kBCKqrm586FB7BcafHmGFrWhT3E\nLOUYkOV/gADT2hVDCrkPosg7Wb6ND9/mhCVVhf4hLGRh\n-----END CERTIFICATE-----\n"

// todo: implement this for specific tests
//type mockCmixNet struct {
//	testingInterface interface{}
//	instance         *network.Instance
//}
//
//func (m mockCmixNet) Connect(ndf *ndf.NetworkDefinition) error {
//	return nil
//}
//
//func (m mockCmixNet) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
//	//TODO implement me
//	return nil, nil
//}
//
//func (m mockCmixNet) GetMaxMessageLength() int {
//	//TODO implement me
//	return 4096
//}
//
//func (m mockCmixNet) Send(recipient *id.ID, fingerprint format.Fingerprint, service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
//	//TODO implement me
//	return 0, ephemeral.Id{}, nil
//}
//
//func (m mockCmixNet) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
//	//TODO implement me
//	return 0, nil, nil
//}
//
//func (m mockCmixNet) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) RemoveIdentity(id *id.ID) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) GetIdentity(get *id.ID) (identity.TrackedID, error) {
//	//TODO implement me
//	return identity.TrackedID{}, nil
//}
//
//func (m mockCmixNet) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint, mp message.Processor) error {
//	//TODO implement me
//	return nil
//}
//
//func (m mockCmixNet) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) DeleteClientFingerprints(identity *id.ID) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) AddService(clientID *id.ID, newService message.Service, response message.Processor) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) DeleteService(clientID *id.ID, toDelete message.Service, processor message.Processor) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) DeleteClientService(clientID *id.ID) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) TrackServices(tracker message.ServicesTracker) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) CheckInProgressMessages() {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) IsHealthy() bool {
//	//TODO implement me
//	return true
//}
//
//func (m mockCmixNet) WasHealthy() bool {
//	//TODO implement me
//	return true
//}
//
//func (m mockCmixNet) AddHealthCallback(f func(bool)) uint64 {
//	//TODO implement me
//	return 0
//}
//
//func (m mockCmixNet) RemoveHealthCallback(u uint64) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) HasNode(nid *id.ID) bool {
//	//TODO implement me
//	return true
//}
//
//func (m mockCmixNet) NumRegisteredNodes() int {
//	//TODO implement me
//	return 0
//}
//
//func (m mockCmixNet) TriggerNodeRegistration(nid *id.ID) {
//	//TODO implement me
//	return
//}
//
//func (m mockCmixNet) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) error {
//	//TODO implement me
//	return nil
//}
//
//func (m mockCmixNet) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
//	//TODO implement me
//	return nil
//}
//
//func (m mockCmixNet) SendToAny(sendFunc func(host *connect2.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
//	//TODO implement me
//	return nil, nil
//}
//
//func (m mockCmixNet) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
//	return nil, nil
//}
//
//func (m mockCmixNet) SetGatewayFilter(f gateway.Filter) {
//	return
//}
//
//func (m mockCmixNet) GetHostParams() connect2.HostParams {
//	return connect2.HostParams{}
//}
//
//func (m mockCmixNet) GetAddressSpace() uint8 {
//	return 0
//}
//
//func (m mockCmixNet) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
//	return nil, nil
//}
//
//func (m mockCmixNet) UnregisterAddressSpaceNotification(tag string) {
//	return
//}
//
//func (m *mockCmixNet) GetInstance() *network.Instance {
//	if m.instance == nil {
//		commsManager := connect2.NewManagerTesting(m.testingInterface)
//
//		instanceComms := &connect2.ProtoComms{
//			Manager: commsManager,
//		}
//
//		def := getNDF(m.testingInterface)
//
//		thisInstance, err := network.NewInstanceTesting(instanceComms, def, def, nil, nil, m.testingInterface)
//		if err != nil {
//			panic(err)
//		}
//
//		m.instance = thisInstance
//	}
//
//	return m.instance
//}
//
//func (m mockCmixNet) GetVerboseRounds() string {
//	return ""
//}

type mockConnect struct {
	sendChan     chan []byte
	responseChan chan []byte
	unregister   chan bool
}

// Closer deletes this Connection's partner.Manager and releases resources
func (mc *mockConnect) Close() error {
	return nil
}

// GetPartner returns the partner.Manager for this Connection
func (mc *mockConnect) GetPartner() partner.Manager {
	return nil
}

// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
func (mc *mockConnect) SendE2E(mt catalog.MessageType, payload []byte, params e2e.Params) (
	[]id.Round, ce2e.MessageID, time.Time, error) {
	mc.sendChan <- payload
	return []id.Round{2}, ce2e.MessageID{1}, time.Now(), nil
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
func (mc *mockConnect) RegisterListener(messageType catalog.MessageType,
	newListener receive.Listener) receive.ListenerID {
	go func() {
		select {
		case p := <-mc.responseChan:
			newListener.Hear(receive.Message{Payload: p})
		case <-mc.unregister:
			return
		}
	}()
	return receive.ListenerID{}
}

// Unregister listener for E2E reception
func (mc *mockConnect) Unregister(listenerID receive.ListenerID) {
	mc.unregister <- true
}

func getNDF(t interface{}) *ndf.NetworkDefinition {
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
		Registration: ndf.Registration{
			EllipticPubKey: "/WRtT+mDZGC3FXQbvuQgfqOonAjJ47IKE0zhaGTQQ70=",
		},
		Gateways: []ndf.Gateway{
			{
				ID:             id.NewIdFromString("GW1", id.Gateway, t).Bytes(),
				Address:        "0.0.0.0:11420",
				TlsCertificate: pub,
				Bin:            region.NorthernAfrica,
			},
		}, Nodes: []ndf.Node{
			{
				ID:             id.NewIdFromString("NODE1", id.Gateway, t).Bytes(),
				Address:        "0.0.0.0:11420",
				TlsCertificate: pub,
			},
		},
	}
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

func TestSimple_Write(t *testing.T) {
	/* Set up testing data */
	//grp := getGroup()
	//rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)

	// Create IDs
	// myID := id.NewIdFromString("zezima", id.User, t)
	//partnerID := id.NewIdFromString("swampman", id.User, t)

	// Generate private and public keys
	//myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
	//	grp, rng.GetStream())
	//_ = dh.GeneratePublicKey(myPrivKey, grp)

	//partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength,
	//	grp, rng.GetStream())
	//partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)

	//partner := contact.Contact{
	//	ID:             partnerID,
	//	DhPubKey:       partnerPubKey,
	//	OwnershipProof: nil,
	//	Facts:          nil,
	//}
	//connParams := connect.GetDefaultParams()
	//c, err := connect.Connect(partner, myID, myPrivKey, rng, grp, &mockCmixNet{}, connParams)
	//if err != nil {
	//	t.Fatalf("Failed to create connection: %+v", err)
	//}
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte, 1),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := []byte("hello from me")
	bytesWritten, err := ss.Write(data)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	}
	if bytesWritten != len(data) {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", len(data), bytesWritten)
	}
	timeout := time.Tick(time.Second)
	select {
	case p := <-mc.sendChan:
		t.Logf("Received payload %+v over sendChan", p)
		if bytes.Compare(p, data) != 0 {
			t.Errorf("Did not receive expected bytes\n\tExpected: %+v\n\tReceived: %+v\n", data, p)
		}
	case <-timeout:
		t.Errorf("Timed out waiting for send")
	}
}

func TestSimple_Read_sameSize(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := make([]byte, 32)
	n0, err := rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	time.Sleep(time.Second)
	receivedData := make([]byte, 32)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != n0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", n0, n)
	} else if bytes.Compare(receivedData, data) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data, receivedData)
	}
}

func TestSimple_Read_larger(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := make([]byte, 32)
	n0, err := rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	time.Sleep(time.Second)
	receivedData := make([]byte, 64)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != n0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", n0, n)
	} else if bytes.Compare(receivedData[:32], data) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data, receivedData)
	}
}

func TestSimple_Read_smaller(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := make([]byte, 32)
	_, err = rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	time.Sleep(time.Second)
	l := len(data) / 2
	receivedData := make([]byte, l)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != l {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedData, data[:l]) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[:l], receivedData)
	}

	receivedDataRest := make([]byte, l)
	n, err = ss.Read(receivedDataRest)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != l {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedDataRest, data[l:]) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[l:], receivedDataRest)
	}
}
