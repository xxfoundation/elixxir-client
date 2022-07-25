////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"io"
	"math/rand"
	"testing"
	"time"

	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// Base64 encoded dh private key using a pre-seeded prng
// 	prng := NewPrng(42)
const dhPrivKeyEnc = `U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVLf15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAg/YdN1vAK0HfT5GSnhj9qeb4LlTnSOgeeeS71v40zcuoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44bC6+uiBuCp1EQikLtPJA8qkNGWnhiBhaXiu0M48bE8657w+BJW1cS/v2+DBAoh+EA2s0tiF9pLLYH2gChHBxwceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGFJiUf980RBDtTBFgI/qONXa2/tJ/+JdLrAyv2a0FaSsTYZ5ziWTf3Hno1TQ3NmHP1m10/sHhuJSRq3I25LdSFikM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDohdMmDx3peCg9Sgmjb5bCCUF0bj7U2mRqmui0+ntPw6ILr6GnXtMnqGuLDDmvHP0rO1EhnqeVM6v0SNLEedMmB1M5BZFMjMHPCdo54Okp0C`

const searchMessageEnc = `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`
const dhKeyEnc = `hQj4FKeaQDt34eO/BQe5OSk571WMQBu4YNqMQ0TFeuzjzCM9lLMuOsQhcyjZcIo4qanfv0egkwW7YgSAAt+WaOc6S1rzbxidb6v00KCEVkXAl3NSYk3ms8Jp0uzejOKEI6I3TCjX+71GZ6304xRKQSXDnuYG7ypvx+tYgj6EWx/sh5DsdIyT0KVxo9c/msT0cT6Fq1Hz1S3TkQc2dLcamn86gtKs4cSNuG/b7HFvyKSCK4KqQ2wrQWhiJCh0L6UhXEqTVwyoVgjH8pA/R+eTIE3QL4rDPy+gFdaZOVbr2jOOEvgYTJxWU7VPygXkPtTTHL8eD5IXoj9b46ll6CAHC5xwkCfiqO+HLE2Cg6nVmIq7eCZOLH1WnrYcjv7TQrYNCdb2hGDZznbI2wVswiIW7CilHaKH5KmiVfZG3cPKfwcTATRFR5oNVcav4wikObzB7gAmP57lGTwF1WqfLlGAxDF1R+Qiljfp1U1f0s7MBjG62gFwxpxs4AgA8lZDnvlRgeM1P0zsaypUbrHXFhu6`
const lookupRequestEnc = `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`

func newTestManager(t *testing.T) (*Manager, *testNetworkManager) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		t.Fatalf("Failed to initialize store %v", err)
	}

	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	stream := rngGen.GetStream()
	privKey, err := rsa.GenerateKey(stream, 1024)
	stream.Close()

	// Create our Manager object
	tnm := newTestNetworkManager(t)
	m := &Manager{
		user: mockE2e{
			grp:     getGroup(),
			events:  event.NewEventManager(),
			rng:     rngGen,
			kv:      kv,
			network: tnm,
			t:       t,
			key:     privKey,
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
	_, err = m.comms.AddHost(udID, netDef.UDB.Address,
		[]byte(netDef.UDB.Cert), params)
	if err != nil {
		t.Fatalf("User Discovery host " +
			"object could not be constructed.")
	}

	udContact, err := m.GetContact()
	if err != nil {
		t.Fatalf("Failed to get contact: %v", err)
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
	key     *rsa.PrivateKey
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
