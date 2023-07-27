////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package pickup

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/pickup/store"
	"gitlab.com/elixxir/client/v4/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

func newManager(t *testing.T) *pickup {
	session := storage.InitTestingSession(t)

	unchecked, err := store.NewUncheckedStore(session.GetKV())
	if err != nil {
		t.Errorf("Failed to make new UncheckedRoundStore: %+v", err)
	}

	instance := &MockRoundGetter{
		topology: [][]byte{
			id.NewIdFromString("gateway0", id.Gateway, t).Bytes(),
			id.NewIdFromString("gateway1", id.Gateway, t).Bytes(),
			id.NewIdFromString("gateway2", id.Gateway, t).Bytes(),
			id.NewIdFromString("gateway3", id.Gateway, t).Bytes(),
		},
	}

	testManager := &pickup{
		params:              GetDefaultParams(),
		session:             session,
		rng:                 fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		instance:            instance,
		lookupRoundMessages: make(chan roundLookup),
		messageBundles:      make(chan message.Bundle),
		unchecked:           unchecked,
	}

	return testManager
}

type MockRoundGetter struct {
	topology [][]byte
}

func (mrg *MockRoundGetter) GetRound(rid id.Round) (*pb.RoundInfo, error) {
	return &pb.RoundInfo{
		ID:       uint64(rid),
		Topology: mrg.topology,
	}, nil
}

// Build ID off of this string for expected gateway that will be returned over
// mock comm
const (
	ReturningGateway = "GetMessageRequest"
	FalsePositive    = "FalsePositive"
	PayloadMessage   = "Payload"
	ErrorGateway     = "Error"
)

type mockMessageRetrievalComms struct {
	testingSignature *testing.T
}

func (mmrc *mockMessageRetrievalComms) AddHost(_ *id.ID, _ string, _ []byte,
	_ connect.HostParams) (host *connect.Host, err error) {
	host, _ = mmrc.GetHost(nil)
	return host, nil
}

func (mmrc *mockMessageRetrievalComms) RemoveHost(_ *id.ID) {
}

func (mmrc *mockMessageRetrievalComms) GetHost(hostId *id.ID) (*connect.Host, bool) {
	p := connect.GetDefaultHostParams()
	p.MaxRetries = 0
	p.AuthEnabled = false
	h, _ := connect.NewHost(hostId, "0.0.0.0", []byte(""), p)
	return h, true
}

// RequestMessages returns differently based on the host ID.
// ReturningGateway returns a happy path response, in which there is a message.
// FalsePositive returns a response in which there were no messages in the
// round.
// ErrorGateway returns an error on the mock comm.
// Any other ID returns default no round errors
func (mmrc *mockMessageRetrievalComms) RequestMessages(host *connect.Host,
	_ *pb.GetMessages) (*pb.GetMessagesResponse, error) {
	payloadMsg := []byte(PayloadMessage)
	payload := make([]byte, 256)
	copy(payload, payloadMsg)
	testSlot := &pb.Slot{
		PayloadA: payload,
		PayloadB: payload,
	}

	// If we are the requesting on the returning gateway, return a mock response
	returningGateway := id.NewIdFromString(
		ReturningGateway, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(returningGateway) {
		return &pb.GetMessagesResponse{
			Messages: []*pb.Slot{testSlot},
			HasRound: true,
		}, nil
	}

	// Return an empty message structure (i.e. a false positive in the bloom
	// filter)
	falsePositive := id.NewIdFromString(
		FalsePositive, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(falsePositive) {
		return &pb.GetMessagesResponse{
			Messages: []*pb.Slot{},
			HasRound: true,
		}, nil
	}

	// Return a mock error
	errorGateway := id.NewIdFromString(
		ErrorGateway, id.Gateway, mmrc.testingSignature)
	if host.GetId().Cmp(errorGateway) {
		return &pb.GetMessagesResponse{}, errors.Errorf("Connection error")
	}

	return nil, nil
}

func (mmrc *mockMessageRetrievalComms) RequestBatchMessages(host *connect.Host, req *pb.GetMessagesBatch) (*pb.GetMessagesResponseBatch, error) {
	ret := make([]*pb.GetMessagesResponse, len(req.GetRequests()))
	for i, mreq := range req.GetRequests() {
		targetId, err := id.Unmarshal(mreq.Target)
		if err != nil {

		}
		h, err := connect.NewHost(targetId, "0.0.0.0", nil, connect.GetDefaultHostParams())
		if err != nil {

		}
		ret[i], err = mmrc.RequestMessages(h, mreq)
	}
	return &pb.GetMessagesResponseBatch{
		Results: ret,
		Errors:  make([]string, len(ret)),
	}, nil
}

func newTestBackoffTable(face interface{}) [cappedTries]time.Duration {
	switch face.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf(
			"newTestBackoffTable is restricted to testing only. Got %T", face)
	}

	var backoff [cappedTries]time.Duration
	for i := 0; i < cappedTries; i++ {
		backoff[uint64(i)] = 1 * time.Millisecond
	}

	return backoff

}

func getNDF() *ndf.NetworkDefinition {
	cert := testkeys.GetNodeCert()
	nodeID := id.NewIdFromBytes([]byte("gateway"), &testing.T{})
	return &ndf.NetworkDefinition{
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
				ID:             nodeID.Bytes(),
				Address:        "",
				TlsCertificate: string(cert),
			},
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
