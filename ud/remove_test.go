package ud

import (
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/messages"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

var genericGroup = cyclic.NewGroup(
	large.NewIntFromString("9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48"+
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F"+
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5"+
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2"+
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41"+
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE"+
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15"+
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B", 16),
	large.NewIntFromString("5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613"+
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4"+
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472"+
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5"+
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA"+
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71"+
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0"+
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7", 16))

type testRFC struct{}

func (rFC *testRFC) SendDeleteMessage(host *connect.Host, message *messages.AuthenticatedMessage) (*messages.Ack, error) {
	return &messages.Ack{}, nil
}

func TestRemoveFact(t *testing.T) {
	c, err := client.NewClientComms(&id.DummyUser, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	h, err := connect.NewHost(&id.DummyUser, "address", nil, connect.GetDefaultHostParams())
	if err != nil {
		t.Fatal(err)
	}

	rng := csprng.NewSystemRNG()
	cpk, err := rsa.GenerateKey(rng, 2048)
	if err != nil {
		t.Fatal(err)
	}

	m := Manager{
		comms:   c,
		host:    h,
		privKey: cpk,
	}

	f := fact.Fact{
		Fact: "testing",
		T:    2,
	}

	trfc := testRFC{}

	err = m.removeFact(f, &trfc)
	if err != nil {
		t.Fatal(err)
	}
}
