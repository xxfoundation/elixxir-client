////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	commNetwork "gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/chacha"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"testing"
	"time"
)

const (
	privKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA8/RbJWR9kD8IR3YOAqly52fSgOEPCyObHwsENaxSBDPIusWJ
WKB9SZR5+SdQfd1MmOaIm0aouxrIKdZ2pYsc8li8V+xZhbQgYX3dQb/g3lQBY7ii
QLcJiApKLh6Sl/DbOBmrSCZvNOddwcX2B+ZDuMlQpIS0k9I3Ner49l4lML3MkBvg
uPxDMfXysMzwNovbLWezRjYe7R50EWf86pq8Ekpv+4CbcASOp46ELsRBSNVqZaVM
E+3jT4H/poUlfkFibYkWRGwYEjIFUvjoy8LTYyDNNP1skgl/CqYFpOu7B6Y2DMa7
smTTeGqpLVNQS9Ijr/HxtWhtreitwqR/0ewOMQIDAQABAoIBAQDxobnZ2qQoCNbZ
eUw9RLtEC2jMMJ8m6FiQMeg0hX8jHGuY22nD+ArAo6kAqPkoAdcJp2XtbtpXoRpb
nkochCLixBOhfr/ZF+Xuyq0pn7VKYaiSrmE/ekydi5uX/L40cuOfuIUXzMHfg78w
3DRp9KBlWjlfCvaVZ+U5qYh49h0eHSF0Le9Q6+gAZ/FCGfLYHI+hpmMK+8QlXvD4
XVnjTEH8dUGUarVGxIw6p0ZsF7T6kYgPHG5e2zc6roIqfOWBG4MBkkYUdPECqY1O
sHbZl5TVUK8GdYhX+U3dnCmC4L96n4djVEGQx68fQB6rJ2WE2VPlpDpj+M4wenPX
MpB02ftBAoGBAP08KOnGrNz40nZ6yfVSRBfkyJHsXAV5XTHtw2u7YSPSXN9tq+7Z
AIVRaO9km2SxWfNDdkEge0xmQjKe/1CkjiwqBBjsDI6P1yYQIReoXndSAZ4JmS8P
6IzdDpv4vDjmC55Y+c5+uFQn8+1zdeYHwQYie+5LxsAKQxo1wbaNaN6JAoGBAPae
QWhZbiEVkftznrPpfAW4Fl/ZlCAIonvn6uYXM/1TMDGqPIOZyfYB6BIFjkdT9aHP
ZhZtFgWNnAyta37EM+FGnDtBTmFJ3tl4gqZWLIK6T2csinrDsdv/s7VpduB0yAE0
sfWuRZoBfEpUof37TS//YR6Ibm/G0IS8LnrSMIhpAoGAYKluFI45vb9c1szX+kSE
qXoy9UB7f7trz3sqdRz5X2sU+FQspOdAQ6NnormMd0sbQrglk4aKigcejaQTYPzv
J/yBw+GWiXRuc6EEgLtME8/Bvkl7p3MzGVHoGbFAZ5eoJ7Fe6WuFgNofSiwgfMXI
8EaJd9SE8Rj5tC+A2eXwecECgYAxXv05Jq4lcWwIKt1apyNtAa15AtXkk9XzeDpO
VdbSoBTF3I7Aycjktvz+np4dKXHDMwH8+1mtQuw6nX0no5+/OaONOUW3tFIotzdw
lU/T2/iJbyFJ8mNo54fSiYqC5N4lX6dAx+KnMiTvvIGxlt2c/kMzGZ0CQ4r7B7FG
ZU3SAQKBgQCxE34846J4kH6jRsboyZVkdDdzXQ+NeICJXcaHM2okjnT50IG6Qpwd
0yPXN6xvYW5L+FVb80NfD1y8LkmBerNEMpcwwDL1ZhgiKWQmITESphnYpm3GV9pe
1vIMaHV6GeX+q/RcLu2kU4hJbH6HDRJxtdkmw/gdSo9vphDgB6qALw==
-----END RSA PRIVATE KEY-----`
	cert = `-----BEGIN CERTIFICATE-----
MIIDbDCCAlSgAwIBAgIJAOUNtZneIYECMA0GCSqGSIb3DQEBBQUAMGgxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQx
GzAZBgNVBAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJp
cDAeFw0xOTAzMDUxODM1NDNaFw0yOTAzMDIxODM1NDNaMGgxCzAJBgNVBAYTAlVT
MRMwEQYDVQQIDApDYWxpZm9ybmlhMRIwEAYDVQQHDAlDbGFyZW1vbnQxGzAZBgNV
BAoMElByaXZhdGVncml0eSBDb3JwLjETMBEGA1UEAwwKKi5jbWl4LnJpcDCCASIw
DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPP0WyVkfZA/CEd2DgKpcudn0oDh
Dwsjmx8LBDWsUgQzyLrFiVigfUmUefknUH3dTJjmiJtGqLsayCnWdqWLHPJYvFfs
WYW0IGF93UG/4N5UAWO4okC3CYgKSi4ekpfw2zgZq0gmbzTnXcHF9gfmQ7jJUKSE
tJPSNzXq+PZeJTC9zJAb4Lj8QzH18rDM8DaL2y1ns0Y2Hu0edBFn/OqavBJKb/uA
m3AEjqeOhC7EQUjVamWlTBPt40+B/6aFJX5BYm2JFkRsGBIyBVL46MvC02MgzTT9
bJIJfwqmBaTruwemNgzGu7Jk03hqqS1TUEvSI6/x8bVoba3orcKkf9HsDjECAwEA
AaMZMBcwFQYDVR0RBA4wDIIKKi5jbWl4LnJpcDANBgkqhkiG9w0BAQUFAAOCAQEA
neUocN4AbcQAC1+b3To8u5UGdaGxhcGyZBlAoenRVdjXK3lTjsMdMWb4QctgNfIf
U/zuUn2mxTmF/ekP0gCCgtleZr9+DYKU5hlXk8K10uKxGD6EvoiXZzlfeUuotgp2
qvI3ysOm/hvCfyEkqhfHtbxjV7j7v7eQFPbvNaXbLa0yr4C4vMK/Z09Ui9JrZ/Z4
cyIkxfC6/rOqAirSdIp09EGiw7GM8guHyggE4IiZrDslT8V3xIl985cbCxSxeW1R
tgH4rdEXuVe9+31oJhmXOE9ux2jCop9tEJMgWg7HStrJ5plPbb+HmjoX3nBO04E5
6m52PyzMNV+2N21IPppKwA==
-----END CERTIFICATE-----
`
)

// Creates new registrar for testing.
func makeTestRegistrar(mockComms *MockClientComms, t *testing.T) *registrar {
	connect.TestingOnlyDisableTLS = true

	session := storage.InitTestingSession(t)
	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(gateway.DefaultPoolParams(), rngGen,
		getNDF(), newMockManager(), session, nil)
	if err != nil {
		t.Fatalf("Failed to create new sender: %+v", err)
	}

	nodeChan := make(chan commNetwork.NodeGateway, InputChanLen)

	r, err := LoadRegistrar(
		session, sender, mockComms, rngGen, nodeChan, func() int { return 100 })
	if err != nil {
		t.Fatalf("Failed to create new registrar: %+v", err)
	}

	return r.(*registrar)
}

///////////////////////////////////////////////////////////////////////////////
///////////////// Mock Sender Interface ///////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockSender struct{}

func (m mockSender) SendToAny(
	sendFunc func(host *connect.Host) (interface{}, error),
	stop *stoppable.Single) (interface{}, error) {

	return sendFunc(nil)

	// implement this one
}

func (m mockSender) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockSender) UpdateNdf(ndf *ndf.NetworkDefinition) {
	//TODO implement me
	panic("implement me")
}

func (m mockSender) SetGatewayFilter(f gateway.Filter) {
	//TODO implement me
	panic("implement me")
}

func (m mockSender) GetHostParams() connect.HostParams {
	//TODO implement me
	panic("implement me")
}

///////////////////////////////////////////////////////////////////////////////
///////////////// Mock storage.session Interface //////////////////////////////
///////////////////////////////////////////////////////////////////////////////

type mockSession struct {
	isPrecanned     bool
	privKey         *rsa.PrivateKey
	timeStamp       time.Time
	salt            []byte
	transmissionSig []byte
}

func (m mockSession) GetCmixGroup() *cyclic.Group {
	return nil
}

func (m mockSession) GetKV() *versioned.KV {
	return nil
}

func (m mockSession) GetTransmissionID() *id.ID {
	return nil
}

func (m mockSession) IsPrecanned() bool {
	return m.isPrecanned
}

func (m mockSession) GetTransmissionRSA() *rsa.PrivateKey {
	return m.privKey
}

func (m mockSession) GetRegistrationTimestamp() time.Time {
	return m.timeStamp
}

func (m mockSession) GetTransmissionSalt() []byte {
	return m.salt
}

func (m mockSession) GetTransmissionRegistrationValidationSignature() []byte {
	return m.transmissionSig
}

///////////////////////////////////////////////////////////////////////////////
///////////////// Mock Comms Interface ///////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

// Mock client comms object adhering to RegisterNodeCommsInterface for testing.
type MockClientComms struct {
	rsaPrivKey *rsa.PrivateKey
	dhPrivKey  *cyclic.Int
	rand       csprng.Source
	grp        *cyclic.Group
	secret     []byte
	t          *testing.T
}

func (m *MockClientComms) BatchNodeRegistration(_ *connect.Host, message *pb.SignedClientBatchKeyRequest) (*pb.SignedBatchKeyResponse, error) {
	resp := make([]*pb.SignedKeyResponse, len(message.Targets))
	for i := range message.Targets {
		part, err := m.SendRequestClientKeyMessage(nil, &pb.SignedClientKeyRequest{
			ClientKeyRequest:          message.ClientKeyRequest,
			ClientKeyRequestSignature: message.ClientKeyRequestSignature,
			Target:                    message.Targets[i],
		})
		if err != nil {
			return nil, err
		}
		resp[i] = part
	}
	return &pb.SignedBatchKeyResponse{
		SignedKeys: resp,
	}, nil
}

func (m *MockClientComms) GetHost(_ *id.ID) (*connect.Host, bool) {
	return &connect.Host{}, true
}

// SendRequestClientKeyMessage mocks up the network's response to a
// client's request to register and sends that data back
// to the caller.
func (m *MockClientComms) SendRequestClientKeyMessage(_ *connect.Host,
	request *pb.SignedClientKeyRequest) (*pb.SignedKeyResponse, error) {

	// Parse serialized data into message
	msg := &pb.ClientKeyRequest{}
	err := proto.Unmarshal(request.ClientKeyRequest, msg)
	if err != nil {
		m.t.Fatalf("Couldn't parse client key request: %v", err)
	}

	// Parse internal message
	clientTransmissionConfirmation := &pb.ClientRegistrationConfirmation{}
	err = proto.Unmarshal(msg.ClientTransmissionConfirmation.
		ClientRegistrationConfirmation, clientTransmissionConfirmation)
	if err != nil {
		m.t.Fatalf("Couldn't parse client registration confirmation: %v", err)
	}

	// Define hashing algorithm
	opts := rsa.NewDefaultOptions()
	opts.Hash = hash.CMixHash
	h := opts.Hash.New()

	// Extract RSA pubkey
	clientRsaPub := clientTransmissionConfirmation.RSAPubKey
	// Assemble client public key into rsa.PublicKey
	userPublicKey, err := rsa.LoadPublicKeyFromPem([]byte(clientRsaPub))
	if err != nil {
		m.t.Fatalf("Failed to load public key: %+v", err)
	}

	// Parse user ID
	userId, err := xx.NewID(userPublicKey, msg.GetSalt(), id.User)
	if err != nil {
		m.t.Fatalf("Failed to generate user id: %+v", err)
	}
	// Construct client key
	h.Reset()
	h.Write(userId.Bytes())
	h.Write(m.secret)
	clientKey := h.Sum(nil)

	// Parse client public key
	clientDHPub := m.grp.NewIntFromBytes(msg.GetClientDHPubKey())

	// Generate session key
	h.Reset()
	sessionKey := registration.GenerateBaseKey(m.grp, clientDHPub,
		m.dhPrivKey, h)

	// Encrypt the client key using the session key
	encryptedClientKey, err := chacha.Encrypt(sessionKey.Bytes(), clientKey,
		m.rand)
	if err != nil {
		m.t.Fatalf("Unable to encrypt key: %v", err)
	}

	// fixme: testing session key does not match what's generating in client
	h.Reset()
	encryptedHMac := registration.CreateClientHMAC(sessionKey.Bytes(),
		encryptedClientKey,
		opts.Hash.New)

	serverDhPub := m.grp.ExpG(m.dhPrivKey, m.grp.NewInt(1))

	keyResponse := &pb.ClientKeyResponse{
		NodeDHPubKey:           serverDhPub.Bytes(),
		EncryptedClientKey:     encryptedClientKey,
		EncryptedClientKeyHMAC: encryptedHMac,
	}

	serializedResponse, err := proto.Marshal(keyResponse)
	if err != nil {
		m.t.Fatalf("Send error: %v", err)
	}

	// Hash the response
	h.Reset()
	h.Write(serializedResponse)
	hashed := h.Sum(nil)

	// Sign the nonce
	signed, err := rsa.Sign(m.rand, m.rsaPrivKey, opts.Hash, hashed, opts)
	if err != nil {
		m.t.Fatalf("Failed to sign a request (as mock gateway): %+v", err)
	}

	return &pb.SignedKeyResponse{
		KeyResponse:                serializedResponse,
		KeyResponseSignedByGateway: &messages.RSASignature{Signature: signed},
		ClientGatewayKey:           m.grp.NewInt(52).Bytes(),
	}, nil
}

///////////////////////////////////////////////////////////////////////////////
///////////////// Mock Host Interface ///////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////

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

func getGroup() *cyclic.Group {
	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642"+
			"F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757"+
			"264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F"+
			"9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091E"+
			"B51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D"+
			"0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D3"+
			"92145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A"+
			"2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7"+
			"995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
		large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E2480"+
			"9670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D"+
			"1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A33"+
			"8661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361"+
			"C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B28"+
			"5DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD929"+
			"59859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D83"+
			"2186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8"+
			"B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16),
	)

	return e2eGrp

}
