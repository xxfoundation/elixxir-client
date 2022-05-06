///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"bytes"
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Tests that manager.MakeGroup adds a group and returns the expected status.
func Test_manager_MakeGroup(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	memberIDs, members, dkl := addPartners(m, t)
	name := []byte("groupName")
	message := []byte("Invite message.")

	g, _, status, err := m.MakeGroup(memberIDs, name, message)
	if err != nil {
		t.Errorf("MakeGroup() returned an error: %+v", err)
	}

	if status != AllSent {
		t.Errorf("MakeGroup() did not return the expected status."+
			"\nexpected: %s\nreceived: %s", AllSent, status)
	}

	_, exists := m.gs.Get(g.ID)
	if !exists {
		t.Errorf("Failed to get group %#v.", g)
	}

	if !reflect.DeepEqual(members, g.Members) {
		t.Errorf("New group does not have expected membership."+
			"\nexpected: %s\nreceived: %s", members, g.Members)
	}

	if !reflect.DeepEqual(dkl, g.DhKeys) {
		t.Errorf("New group does not have expected DH key list."+
			"\nexpected: %#v\nreceived: %#v", dkl, g.DhKeys)
	}

	if !g.ID.Cmp(g.ID) {
		t.Errorf("New group does not have expected ID."+
			"\nexpected: %s\nreceived: %s", g.ID, g.ID)
	}

	if !bytes.Equal(name, g.Name) {
		t.Errorf("New group does not have expected name."+
			"\nexpected: %q\nreceived: %q", name, g.Name)
	}

	if !bytes.Equal(message, g.InitMessage) {
		t.Errorf("New group does not have expected message."+
			"\nexpected: %q\nreceived: %q", message, g.InitMessage)
	}
}

// Error path: make sure an error and the correct status is returned when the
// message is too large.
func Test_manager_MakeGroup_MaxMessageSizeError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	expectedErr := fmt.Sprintf(
		maxInitMsgSizeErr, MaxInitMessageSize+1, MaxInitMessageSize)

	_, _, status, err := m.MakeGroup(nil, nil, make([]byte, MaxInitMessageSize+1))
	if err == nil || err.Error() != expectedErr {
		t.Errorf("MakeGroup() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if status != NotSent {
		t.Errorf("MakeGroup() did not return the expected status."+
			"\nexpected: %s\nreceived: %s", NotSent, status)
	}
}

// Error path: make sure an error and the correct status is returned when the
// membership list is too small.
func Test_manager_MakeGroup_MembershipSizeError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, 10, 0, nil, t)
	expectedErr := fmt.Sprintf(
		maxMembersErr, group.MaxParticipants+1, group.MaxParticipants)

	_, _, status, err := m.MakeGroup(make([]*id.ID, group.MaxParticipants+1),
		nil, []byte{})
	if err == nil || err.Error() != expectedErr {
		t.Errorf("MakeGroup() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if status != NotSent {
		t.Errorf("MakeGroup() did not return the expected status."+
			"\nexpected: %s\nreceived: %s", NotSent, status)
	}
}

// Error path: make sure an error and the correct status is returned when adding
// a group failed because the user is a part of too many groups already.
func Test_manager_MakeGroup_AddGroupError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManagerWithStore(prng, gs.MaxGroupChats, 0, nil, t)
	memberIDs, _, _ := addPartners(m, t)
	expectedErr := strings.SplitN(joinGroupErr, "%", 2)[0]

	_, _, _, err := m.MakeGroup(memberIDs, []byte{}, []byte{})
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("MakeGroup() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of manager.buildMembership.
func Test_manager_buildMembership(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManager(prng, t)
	memberIDs, expected, expectedDKL := addPartners(m, t)

	membership, dkl, err := m.buildMembership(memberIDs)
	if err != nil {
		t.Errorf("buildMembership() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, membership) {
		t.Errorf("buildMembership() failed to return the expected membership."+
			"\nexpected: %s\nrecieved: %s", expected, membership)
	}

	if !reflect.DeepEqual(expectedDKL, dkl) {
		t.Errorf("buildMembership() failed to return the expected DH key list."+
			"\nexpected: %#v\nrecieved: %#v", expectedDKL, dkl)
	}
}

// Error path: an error is returned when the number of members in the membership
// list is too few.
func Test_manager_buildMembership_MinParticipantsError(t *testing.T) {
	m, _ := newTestManager(rand.New(rand.NewSource(42)), t)
	memberIDs := make([]*id.ID, group.MinParticipants-1)
	expectedErr := fmt.Sprintf(
		minMembersErr, len(memberIDs), group.MinParticipants)

	_, _, err := m.buildMembership(memberIDs)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMembership() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned when the number of members in the membership
// list is too many.
func Test_manager_buildMembership_MaxParticipantsError(t *testing.T) {
	m, _ := newTestManager(rand.New(rand.NewSource(42)), t)
	memberIDs := make([]*id.ID, group.MaxParticipants+1)
	expectedErr := fmt.Sprintf(
		maxMembersErr, len(memberIDs), group.MaxParticipants)

	_, _, err := m.buildMembership(memberIDs)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMembership() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: error returned when a partner cannot be found
func Test_manager_buildMembership_GetPartnerContactError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManager(prng, t)
	memberIDs, _, _ := addPartners(m, t)
	expectedErr := strings.SplitN(getPartnerErr, "%", 2)[0]

	// Replace a partner ID
	memberIDs[len(memberIDs)/2] = id.NewIdFromString("nonPartnerID", id.User, t)

	_, _, err := m.buildMembership(memberIDs)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMembership() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: error returned when a member ID appears twice on the list.
func Test_manager_buildMembership_DuplicateContactError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, _ := newTestManager(prng, t)
	memberIDs, _, _ := addPartners(m, t)
	expectedErr := strings.SplitN(makeMembershipErr, "%", 2)[0]

	// Replace a partner ID
	memberIDs[5] = memberIDs[4]

	_, _, err := m.buildMembership(memberIDs)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("buildMembership() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Test that getPreimages produces unique preimages.
func Test_getPreimages_Unique(t *testing.T) {
	streamGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	n := 100
	idPreimages := make(map[group.IdPreimage]bool, n)
	keyPreimages := make(map[group.KeyPreimage]bool, n)

	for i := 0; i < n; i++ {
		idPreimage, keyPreimage, err := getPreimages(streamGen)
		if err != nil {
			t.Errorf("getPreimages() returned an error: %+v", err)
		}

		if idPreimages[idPreimage] {
			t.Errorf("getPreimages() produced a duplicate idPreimage: %s", idPreimage)
		} else {
			idPreimages[idPreimage] = true
		}

		if keyPreimages[keyPreimage] {
			t.Errorf("getPreimages() produced a duplicate keyPreimage: %s", keyPreimage)
		} else {
			keyPreimages[keyPreimage] = true
		}
	}
}

// Unit test of RequestStatus.String.
func TestRequestStatus_String(t *testing.T) {
	statusCodes := map[RequestStatus]string{
		NotSent:     "NotSent",
		AllFail:     "AllFail",
		PartialSent: "PartialSent",
		AllSent:     "AllSent",
		AllSent + 1: "INVALID STATUS",
	}

	for status, expected := range statusCodes {
		if status.String() != expected {
			t.Errorf("String() failed to return the expected name."+
				"\nexpected: %s\nreceived: %s", expected, status.String())
		}
	}
}

// Unit test of RequestStatus.Message.
func TestRequestStatus_Message(t *testing.T) {
	statusCodes := map[RequestStatus]string{
		NotSent:     "an error occurred before sending any group requests",
		AllFail:     "all group requests failed to send",
		PartialSent: "some group requests failed to send",
		AllSent:     "all groups requests successfully sent",
		AllSent + 1: "INVALID STATUS " + strconv.Itoa(int(AllSent)+1),
	}

	for status, expected := range statusCodes {
		if status.Message() != expected {
			t.Errorf("Message() failed to return the expected message."+
				"\nexpected: %s\nreceived: %s", expected, status.Message())
		}
	}
}

// addPartners returns a list of user IDs and their matching membership and adds
// them as partners.
func addPartners(m *manager, t *testing.T) ([]*id.ID, group.Membership,
	gs.DhKeyList) {
	memberIDs := make([]*id.ID, 10)
	members := group.Membership{m.gs.GetUser()}
	dkl := gs.DhKeyList{}

	for i := range memberIDs {
		// Build member data
		uid := id.NewIdFromUInt(uint64(i), id.User, t)
		dhKey := m.grp.NewInt(int64(i + 42))

		myVariant := sidh.KeyVariantSidhA
		prng := rand.New(rand.NewSource(int64(i + 42)))
		mySIDHPrivKey := util.NewSIDHPrivateKey(myVariant)
		mySIDHPubKey := util.NewSIDHPublicKey(myVariant)
		_ = mySIDHPrivKey.Generate(prng)
		mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

		theirVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
		theirSIDHPrivKey := util.NewSIDHPrivateKey(theirVariant)
		theirSIDHPubKey := util.NewSIDHPublicKey(theirVariant)
		_ = theirSIDHPrivKey.Generate(prng)
		theirSIDHPrivKey.GeneratePublicKey(theirSIDHPubKey)

		// Add to lists
		memberIDs[i] = uid
		members = append(members, group.Member{ID: uid, DhKey: dhKey})
		dkl.Add(dhKey, group.Member{ID: uid, DhKey: dhKey},
			m.grp)

		// Add partner
		_, err := m.e2e.AddPartner(uid, dhKey, dhKey,
			theirSIDHPubKey, mySIDHPrivKey,
			session.GetDefaultParams(),
			session.GetDefaultParams())
		if err != nil {
			t.Errorf("Failed to add partner %d: %+v", i, err)
		}
	}

	return memberIDs, members, dkl
}
