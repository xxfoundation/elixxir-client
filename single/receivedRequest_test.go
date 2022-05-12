////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"gitlab.com/elixxir/client/cmix"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"reflect"
	"testing"
	"time"
)

// Tests that RequestCmix adheres to the cmix.Client interface.
var _ RequestCmix = (cmix.Client)(nil)

func TestRequest_Respond(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	privKey := grp.NewInt(42)
	used := uint32(0)
	payloadChan := make(chan format.Message, 10)
	net := newMockRequestCmix(payloadChan, t)
	r := &Request{
		sender:         id.NewIdFromString("singleUseRequest", id.User, t),
		senderPubKey:   grp.NewInt(42),
		dhKey:          grp.ExpG(privKey, grp.NewInt(1)),
		tag:            "requestTag",
		maxParts:       10,
		used:           &used,
		requestPayload: []byte("test"),
		net:            net,
	}

	payload := []byte("My Response.")
	cMixParams := cmix.GetDefaultCMIXParams()

	_, err := r.Respond(payload, cMixParams, 0)
	if err != nil {
		t.Errorf("Respond returned an error: %+v", err)
	}

	select {
	case ecrMsg := <-payloadChan:
		c := &cypher{
			dhKey:  r.dhKey,
			num:    0,
			newKey: singleUse.NewResponseKey,
			newFp:  singleUse.NewResponseFingerprint,
		}

		decrypted, err := c.decrypt(ecrMsg.GetContents(), ecrMsg.GetMac())
		if err != nil {
			t.Errorf("Failed to decrypt single-use response payload: %+v", err)
			return
		}

		// Unmarshal the cMix message contents to a request message
		responsePart, err := message.UnmarshalResponsePart(decrypted)
		if err != nil {
			t.Errorf("could not unmarshal ResponsePart: %+v", err)
		}

		if !bytes.Equal(payload, responsePart.GetContents()) {
			t.Errorf("Did not receive expected payload."+
				"\nexpected: %q\nreceived: %q",
				payload, responsePart.GetContents())
		}

	case <-time.After(15 * time.Millisecond):
		t.Errorf("Timed out waiting for response.")
	}
}

// Tests that partitionResponse creates a list of message.ResponsePart each with
// the expected contents and part number.
func Test_partitionResponse(t *testing.T) {
	cmixMessageLength := 10

	maxSize := message.NewResponsePart(cmixMessageLength).GetMaxContentsSize()
	maxParts := uint8(10)
	payload := []byte("012345678901234567890123456789012345678901234567890123" +
		"45678901234567890123456789012345678901234567890123456789")
	expectedParts := [][]byte{
		payload[:maxSize],
		payload[maxSize : 2*maxSize],
		payload[2*maxSize : 3*maxSize],
		payload[3*maxSize : 4*maxSize],
		payload[4*maxSize : 5*maxSize],
		payload[5*maxSize : 6*maxSize],
		payload[6*maxSize : 7*maxSize],
		payload[7*maxSize : 8*maxSize],
		payload[8*maxSize : 9*maxSize],
		payload[9*maxSize : 10*maxSize],
	}

	parts := partitionResponse(payload, cmixMessageLength, maxParts)

	for i, part := range parts {
		if part.GetNumParts() != maxParts {
			t.Errorf("Part #%d has wrong numParts.\nexpected: %d\nreceived: %d",
				i, maxParts, part.GetNumParts())
		}
		if int(part.GetPartNum()) != i {
			t.Errorf("Part #%d has wrong part num.\nexpected: %d\nreceived: %d",
				i, i, part.GetPartNum())
		}
		if !bytes.Equal(part.GetContents(), expectedParts[i]) {
			t.Errorf("Part #%d has wrong contents.\nexpected: %q\nreceived: %q",
				i, expectedParts[i], part.GetContents())
		}
	}
}

// Tests that splitPayload splits the payload to match the expected.
func Test_splitPayload(t *testing.T) {
	maxSize := 5
	maxParts := 10
	payload := []byte("012345678901234567890123456789012345678901234567890123" +
		"45678901234567890123456789012345678901234567890123456789")
	expectedParts := [][]byte{
		payload[:maxSize],
		payload[maxSize : 2*maxSize],
		payload[2*maxSize : 3*maxSize],
		payload[3*maxSize : 4*maxSize],
		payload[4*maxSize : 5*maxSize],
		payload[5*maxSize : 6*maxSize],
		payload[6*maxSize : 7*maxSize],
		payload[7*maxSize : 8*maxSize],
		payload[8*maxSize : 9*maxSize],
		payload[9*maxSize : 10*maxSize],
	}

	testParts := splitPayload(payload, maxSize, maxParts)

	if !reflect.DeepEqual(expectedParts, testParts) {
		t.Errorf("splitPayload() failed to correctly split the payload."+
			"\nexpected: %s\nreceived: %s", expectedParts, testParts)
	}
}

////////////////////////////////////////////////////////////////////////////////
// Mock cMix Client                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockRequestCmix struct {
	sendPayload   chan format.Message
	numPrimeBytes int
	instance      *network.Instance
}

func newMockRequestCmix(sendPayload chan format.Message, t *testing.T) *mockRequestCmix {
	instanceComms := &connect.ProtoComms{Manager: connect.NewManagerTesting(t)}
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	thisInstance, err := network.NewInstanceTesting(
		instanceComms, getNDF(), getNDF(), grp, grp, t)
	if err != nil {
		t.Errorf("Failed to create new test instance: %v", err)
	}

	return &mockRequestCmix{
		sendPayload:   sendPayload,
		numPrimeBytes: 97,
		instance:      thisInstance,
	}
}

func (m *mockRequestCmix) GetMaxMessageLength() int {
	msg := format.NewMessage(m.numPrimeBytes)
	return msg.ContentsSize()
}

func (m *mockRequestCmix) Send(_ *id.ID, fp format.Fingerprint,
	_ cmixMsg.Service, payload, mac []byte, _ cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)
	msg.SetContents(payload)
	m.sendPayload <- msg

	return 0, ephemeral.Id{}, nil
}

func (m *mockRequestCmix) GetInstance() *network.Instance {
	return m.instance
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
