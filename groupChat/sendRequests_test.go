////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	sessionImport "gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	util "gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Tests that manager.ResendRequest sends all expected requests successfully.
func Test_manager_ResendRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	expected := &Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
		Created:     g.Created.UnixNano(),
	}

	for i := range g.Members {
		grp := m.getE2eGroup()
		dhKey := grp.NewInt(int64(i + 42))
		pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
		p := sessionImport.GetDefaultParams()
		rng := csprng.NewSystemRNG()
		_, mySidhPriv := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhA, rng)
		theirSidhPub, _ := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhB, rng)
		_, err := m.getE2eHandler().AddPartner(g.Members[i].ID, pubKey, dhKey,
			mySidhPriv, theirSidhPub, p, p)
		if err != nil {
			t.Errorf("Failed to add partner #%d %s: %+v", i, g.Members[i].ID, err)
		}
	}

	_, status, err := m.ResendRequest(g.ID)
	if err != nil {
		t.Errorf("ResendRequest() returned an error: %+v", err)
	}

	if status != AllSent {
		t.Errorf("ResendRequest() failed to return the expected status."+
			"\nexpected: %s\nreceived: %s", AllSent, status)
	}

	if len(m.getE2eHandler().(*testE2eManager).e2eMessages) < len(g.Members)-1 {
		t.Errorf("ResendRequest() failed to send the correct number of requests."+
			"\nexpected: %d\nreceived: %d", len(g.Members)-1,
			len(m.getE2eHandler().(*testE2eManager).e2eMessages))
	}

	for i := 0; i < len(m.getE2eHandler().(*testE2eManager).e2eMessages); i++ {
		msg := m.getE2eHandler().(*testE2eManager).GetE2eMsg(i)

		// Check if the message recipient is a member in the group
		matchesMember := false
		for j, m := range g.Members {
			if msg.Recipient.Cmp(m.ID) {
				matchesMember = true
				g.Members = append(g.Members[:j], g.Members[j+1:]...)
				break
			}
		}
		if !matchesMember {
			t.Errorf("Message %d has recipient ID %s that is not in membership.",
				i, msg.Recipient)
		}

		testRequest := &Request{}
		err = proto.Unmarshal(msg.Payload, testRequest)
		if err != nil {
			t.Errorf("Failed to unmarshal proto message (%d): %+v", i, err)
		}

		if expected.String() != testRequest.String() {
			t.Errorf("Message %d has unexpected payload."+
				"\nexpected: %s\nreceived: %s", i, expected, testRequest)
		}
	}
}

// Error path: an error is returned when no group with the corresponding group
// ID exists.
func Test_manager_ResendRequest_GetGroupError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	expectedErr := strings.SplitN(resendGroupIdErr, "%", 2)[0]

	_, status, err := m.ResendRequest(id.NewIdFromString("invalidID", id.Group, t))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("ResendRequest() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if status != NotSent {
		t.Errorf("ResendRequest() failed to return the expected status."+
			"\nexpected: %s\nreceived: %s", NotSent, status)
	}
}

// Tests that manager.sendRequests sends all expected requests successfully.
func Test_manager_sendRequests(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	expected := &Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
		Created:     g.Created.UnixNano(),
	}

	for i := range g.Members {
		grp := m.getE2eGroup()
		dhKey := grp.NewInt(int64(i + 42))
		pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
		p := sessionImport.GetDefaultParams()
		rng := csprng.NewSystemRNG()
		_, mySidhPriv := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhA, rng)
		theirSidhPub, _ := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhB, rng)
		_, err := m.getE2eHandler().AddPartner(g.Members[i].ID, pubKey, dhKey,
			mySidhPriv, theirSidhPub, p, p)
		if err != nil {
			t.Errorf("Failed to add partner #%d %s: %+v", i, g.Members[i].ID, err)
		}
	}

	_, status, err := m.sendRequests(g)
	if err != nil {
		t.Errorf("sendRequests() returned an error: %+v", err)
	}

	if status != AllSent {
		t.Errorf("sendRequests() failed to return the expected status."+
			"\nexpected: %s\nreceived: %s", AllSent, status)
	}

	if len(m.getE2eHandler().(*testE2eManager).e2eMessages) < len(g.Members)-1 {
		t.Errorf("sendRequests() failed to send the correct number of requests."+
			"\nexpected: %d\nreceived: %d", len(g.Members)-1,
			len(m.getE2eHandler().(*testE2eManager).e2eMessages))
	}

	for i := 0; i < len(m.getE2eHandler().(*testE2eManager).e2eMessages); i++ {
		msg := m.getE2eHandler().(*testE2eManager).GetE2eMsg(i)

		// Check if the message recipient is a member in the group
		matchesMember := false
		for j, m := range g.Members {
			if msg.Recipient.Cmp(m.ID) {
				matchesMember = true
				g.Members = append(g.Members[:j], g.Members[j+1:]...)
				break
			}
		}
		if !matchesMember {
			t.Errorf("Message %d has recipient ID %s that is not in membership.",
				i, msg.Recipient)
		}

		testRequest := &Request{}
		err = proto.Unmarshal(msg.Payload, testRequest)
		if err != nil {
			t.Errorf("Failed to unmarshal proto message (%d): %+v", i, err)
		}

		if expected.String() != testRequest.String() {
			t.Errorf("Message %d has unexpected payload."+
				"\nexpected: %s\nreceived: %s", i, expected, testRequest)
		}
	}
}

// Tests that manager.sendRequests returns the correct status when all sends
// fail.
func Test_manager_sendRequests_SendAllFail(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 1, nil, t)
	expectedErr := fmt.Sprintf(sendRequestAllErr, len(g.Members)-1, "")

	rounds, status, err := m.sendRequests(g)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("sendRequests() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if status != AllFail {
		t.Errorf("sendRequests() failed to return the expected status."+
			"\nexpected: %s\nreceived: %s", AllFail, status)
	}

	if rounds != nil {
		t.Errorf("sendRequests() returned rounds on failure."+
			"\nexpected: %v\nreceived: %v", nil, rounds)
	}

	if len(m.getE2eHandler().(*testE2eManager).e2eMessages) != 0 {
		t.Errorf("sendRequests() sent %d messages when sending should have failed.",
			len(m.getE2eHandler().(*testE2eManager).e2eMessages))
	}
}

// Tests that manager.sendRequests returns the correct status when some sends
// fail.
func Test_manager_sendRequests_SendPartialSent(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 2, nil, t)
	expectedErr := fmt.Sprintf(sendRequestPartialErr, (len(g.Members)-1)/2,
		len(g.Members)-1, "")

	for i := range g.Members {
		grp := m.getE2eGroup()
		dhKey := grp.NewInt(int64(i + 42))
		pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
		p := sessionImport.GetDefaultParams()
		rng := csprng.NewSystemRNG()
		_, mySidhPriv := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhA, rng)
		theirSidhPub, _ := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhB, rng)
		_, err := m.getE2eHandler().AddPartner(g.Members[i].ID, pubKey, dhKey,
			mySidhPriv, theirSidhPub, p, p)
		if err != nil {
			t.Errorf("Failed to add partner #%d %s: %+v", i, g.Members[i].ID, err)
		}
	}

	_, status, err := m.sendRequests(g)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("sendRequests() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if status != PartialSent {
		t.Errorf("sendRequests() failed to return the expected status."+
			"\nexpected: %s\nreceived: %s", PartialSent, status)
	}

	if len(m.getE2eHandler().(*testE2eManager).e2eMessages) != (len(g.Members)-1)/2+1 {
		t.Errorf("sendRequests() sent %d out of %d expected messages.",
			len(m.getE2eHandler().(*testE2eManager).e2eMessages), (len(g.Members)-1)/2+1)
	}
}

// Unit test of manager.sendRequest.
func Test_manager_sendRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	for i := range g.Members {
		grp := m.getE2eGroup()
		dhKey := grp.NewInt(int64(i + 42))
		pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
		p := sessionImport.GetDefaultParams()
		rng := csprng.NewSystemRNG()
		_, mySidhPriv := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhA, rng)
		theirSidhPub, _ := util.GenerateSIDHKeyPair(
			sidh.KeyVariantSidhB, rng)
		_, err := m.getE2eHandler().AddPartner(g.Members[i].ID, pubKey, dhKey,
			mySidhPriv, theirSidhPub, p, p)
		if err != nil {
			t.Errorf("Failed to add partner #%d %s: %+v", i, g.Members[i].ID, err)
		}
	}

	_, err := m.sendRequest(g.Members[0].ID, []byte("request message"))
	if err != nil {
		t.Errorf("sendRequest() returned an error: %+v", err)
	}
	expected := testE2eMessage{
		Recipient: g.Members[0].ID,
		Payload:   []byte("request message"),
	}

	received := m.getE2eHandler().(*testE2eManager).GetE2eMsg(0)

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("sendRequest() did not send the correct message."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Error path: an error is returned when SendE2E fails
func Test_manager_sendRequest_SendE2eError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 1, nil, t)
	expectedErr := strings.SplitN(sendE2eErr, "%", 2)[0]

	recipientID := id.NewIdFromString("memberID", id.User, t)

	grp := m.getE2eGroup()
	dhKey := grp.NewInt(int64(42))
	pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
	p := sessionImport.GetDefaultParams()
	rng := csprng.NewSystemRNG()
	_, mySidhPriv := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(
		sidh.KeyVariantSidhB, rng)
	_, err := m.getE2eHandler().AddPartner(recipientID, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipientID, err)
	}

	_, err = m.sendRequest(recipientID, nil)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("sendRequest() failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of roundIdMap2List.
func Test_roundIdMap2List(t *testing.T) {
	prng := rand.New(rand.NewSource(42))

	// Construct map and expected list
	n := 100
	expected := make([]id.Round, n)
	ridMap := make(map[id.Round]struct{}, n)
	for i := 0; i < n; i++ {
		expected[i] = id.Round(prng.Uint64())
		ridMap[expected[i]] = struct{}{}
	}

	// Create list of IDs from map
	ridList := roundIdMap2List(ridMap)

	// Sort expected and received slices to see if they match
	sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })
	sort.Slice(ridList, func(i, j int) bool { return ridList[i] < ridList[j] })

	if !reflect.DeepEqual(expected, ridList) {
		t.Errorf("roundIdMap2List() failed to return the expected list."+
			"\nexpected: %v\nreceived: %v", expected, ridList)
	}

}
