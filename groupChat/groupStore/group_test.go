////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Unit test of NewGroup.
func TestNewGroup(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	membership := createMembership(prng, 10, t)
	dkl := GenerateDhKeyList(
		membership[0].ID, randCycInt(prng), membership, getGroup())

	expectedGroup := Group{
		Name:        []byte(groupName),
		ID:          id.NewIdFromUInt(uint64(42), id.Group, t),
		Key:         newKey(groupKey),
		IdPreimage:  newIdPreimage(groupIdPreimage),
		KeyPreimage: newKeyPreimage(groupKeyPreimage),
		InitMessage: []byte(initMessage),
		Created:     created,
		Members:     membership,
		DhKeys:      dkl,
	}

	receivedGroup := NewGroup(
		[]byte(groupName),
		id.NewIdFromUInt(uint64(42), id.Group, t),
		newKey(groupKey),
		newIdPreimage(groupIdPreimage),
		newKeyPreimage(groupKeyPreimage),
		[]byte(initMessage),
		expectedGroup.Created,
		membership,
		dkl,
	)

	if !reflect.DeepEqual(receivedGroup, expectedGroup) {
		t.Errorf("NewGroup did not return the expected Group."+
			"\nexpected: %#v\nreceived: %#v", expectedGroup, receivedGroup)
	}
}

// Unit test of Group.DeepCopy.
func TestGroup_DeepCopy(t *testing.T) {
	grp := createTestGroup(rand.New(rand.NewSource(42)), t)

	newGrp := grp.DeepCopy()

	if !reflect.DeepEqual(grp, newGrp) {
		t.Errorf("DeepCopy did not return a copy of the original Group."+
			"\nexpected: %#v\nreceived: %#v", grp, newGrp)
	}

	if &grp.Name[0] == &newGrp.Name[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of Name."+
			"\nexpected: %p\nreceived: %p", &grp.Name[0], &newGrp.Name[0])
	}

	if &grp.ID[0] == &newGrp.ID[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of ID."+
			"\nexpected: %p\nreceived: %p", &grp.ID[0], &newGrp.ID[0])
	}

	if &grp.Key[0] == &newGrp.Key[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of Key."+
			"\nexpected: %p\nreceived: %p", &grp.Key[0], &newGrp.Key[0])
	}

	if &grp.IdPreimage[0] == &newGrp.IdPreimage[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of IdPreimage."+
			"\nexpected: %p\nreceived: %p", &grp.IdPreimage[0], &newGrp.IdPreimage[0])
	}

	if &grp.KeyPreimage[0] == &newGrp.KeyPreimage[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of KeyPreimage."+
			"\nexpected: %p\nreceived: %p", &grp.KeyPreimage[0], &newGrp.KeyPreimage[0])
	}

	if &grp.InitMessage[0] == &newGrp.InitMessage[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of InitMessage."+
			"\nexpected: %p\nreceived: %p", &grp.InitMessage[0], &newGrp.InitMessage[0])
	}

	if &grp.Members[0] == &newGrp.Members[0] {
		t.Errorf("DeepCopy returned a copy of the pointer of Members."+
			"\nexpected: %p\nreceived: %p", &grp.Members[0], &newGrp.Members[0])
	}
}

// Unit test of Group.store.
func TestGroup_store(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	g := createTestGroup(rand.New(rand.NewSource(42)), t)

	err := g.store(kv)
	if err != nil {
		t.Errorf("store returned an error: %+v", err)
	}

	data, err := kv.Get(groupStoreKey(g.ID), groupStoreVersion)
	if err != nil {
		t.Errorf("Failed to get group from storage: %+v", err)
	}

	newGrp, err := DeserializeGroup(data)
	if err != nil {
		t.Errorf("Failed to deserialize group: %+v", err)
	}

	if !reflect.DeepEqual(g, newGrp) {
		t.Errorf("Failed to read correct group from storage."+
			"\nexpected: %#v\nreceived: %#v", g, newGrp)
	}
}

// Unit test of Group.loadGroup.
func Test_loadGroup(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	g := createTestGroup(rand.New(rand.NewSource(42)), t)

	err := g.store(kv)
	if err != nil {
		t.Errorf("store returned an error: %+v", err)
	}

	newGrp, err := loadGroup(g.ID, kv)
	if err != nil {
		t.Errorf("loadGroup returned an error: %+v", err)
	}

	if !reflect.DeepEqual(g, newGrp) {
		t.Errorf("loadGroup failed to return the expected group."+
			"\nexpected: %#v\nreceived: %#v", g, newGrp)
	}
}

// Error path: an error is returned when no group with the ID exists in storage.
func Test_loadGroup_InvalidGroupIdError(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	g := createTestGroup(rand.New(rand.NewSource(42)), t)
	expectedErr := strings.SplitN(kvGetGroupErr, "%", 2)[0]

	_, err := loadGroup(g.ID, kv)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("loadGroup failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of Group.removeGroup.
func Test_removeGroup(t *testing.T) {
	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	g := createTestGroup(rand.New(rand.NewSource(42)), t)

	err := g.store(kv)
	if err != nil {
		t.Errorf("store returned an error: %+v", err)
	}

	err = removeGroup(g.ID, kv)
	if err != nil {
		t.Errorf("removeGroup returned an error: %+v", err)
	}

	foundGrp, err := loadGroup(g.ID, kv)
	if err == nil {
		t.Errorf("loadGroup found group that should have been removed: %#v",
			foundGrp)
	}
}

// Tests that a group that is serialized and deserialized matches the original.
func TestGroup_Serialize_DeserializeGroup(t *testing.T) {
	grp := createTestGroup(rand.New(rand.NewSource(42)), t)

	grpBytes := grp.Serialize()

	newGrp, err := DeserializeGroup(grpBytes)
	if err != nil {
		t.Errorf("DeserializeGroup returned an error: %+v", err)
	}

	if !reflect.DeepEqual(grp, newGrp) {
		t.Errorf("Deserialized group does not match original."+
			"\nexpected: %#v\nreceived: %#v", grp, newGrp)
	}
}

// Tests that a group with nil fields that is serialized and deserialized
// matches the original.
func TestGroup_Serialize_DeserializeGroup_NilGroup(t *testing.T) {
	grp := Group{Members: make(group.Membership, 3)}

	grpBytes := grp.Serialize()

	newGrp, err := DeserializeGroup(grpBytes)
	if err != nil {
		t.Errorf("DeserializeGroup returned an error: %+v", err)
	}

	if !reflect.DeepEqual(grp, newGrp) {
		t.Errorf("Deserialized group does not match original."+
			"\nexpected: %#v\nreceived: %#v", grp, newGrp)
	}
}

// Error path: error returned when the group membership is too small.
func TestDeserializeGroup_DeserializeMembershipError(t *testing.T) {
	grp := Group{}
	grpBytes := grp.Serialize()
	expectedErr := strings.SplitN(membershipErr, "%", 2)[0]

	_, err := DeserializeGroup(grpBytes)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("DeserializeGroup failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

func Test_groupStoreKey(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedKeys := []string{
		"GroupChat/U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
		"GroupChat/15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD",
		"GroupChat/YdN1vAK0HfT5GSnhj9qeb4LlTnSOgeeeS71v40zcuoQD",
		"GroupChat/6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44bC68D",
		"GroupChat/iBuCp1EQikLtPJA8qkNGWnhiBhaXiu0M48bE8657w+AD",
		"GroupChat/W1cS/v2+DBAoh+EA2s0tiF9pLLYH2gChHBxwceeWotwD",
		"GroupChat/wlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGFJiUf980QD",
		"GroupChat/DtTBFgI/qONXa2/tJ/+JdLrAyv2a0FaSsTYZ5ziWTf0D",
		"GroupChat/no1TQ3NmHP1m10/sHhuJSRq3I25LdSFikM8r60LDyicD",
		"GroupChat/hWDxqsBnzqbov0bUqytGgEAsX7KCDohdMmDx3peCg9QD",
	}
	for i, expected := range expectedKeys {
		newID, _ := id.NewRandomID(prng, id.User)

		key := groupStoreKey(newID)

		if key != expected {
			t.Errorf("groupStoreKey did not return the expected key (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, key)
		}

		// fmt.Printf("\"%s\",\n", key)
	}
}

// Unit test of Group.GoString.
func TestGroup_GoString(t *testing.T) {
	grp := createTestGroup(rand.New(rand.NewSource(42)), t)
	grp.Created = grp.Created.UTC()
	expected := "{Name:\"groupName\", " +
		"ID:XMCYoCcs5+sAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAE, " +
		"Key:a2V5AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"IdPreimage:aWRQcmVpbWFnZQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"KeyPreimage:a2V5UHJlaW1hZ2UAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"InitMessage:\"initMessage\", " +
		"Created:" + grp.Created.String() + ", " +
		"Members:{" +
		"Leader: {U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID, 3534334367... in GRP: 6SsQ/HAHUn...}, " +
		"Participants: " +
		"0: {Grcjbkt1IWKQzyvrQsPKJzKFYPGqwGfOpui/RtSrK0YD, 5274380952... in GRP: 6SsQ/HAHUn...}, " +
		"1: {QCxg8d6XgoPUoJo2+WwglBdG4+1NpkaprotPp7T8OiAD, 1628829379... in GRP: 6SsQ/HAHUn...}, " +
		"2: {invD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHAD, 4157513341... in GRP: 6SsQ/HAHUn...}, " +
		"3: {o54Okp0CSry8sWk5e7c05+8KbgHxhU3rX+Qk/vesIQgD, 6317064433... in GRP: 6SsQ/HAHUn...}, " +
		"4: {wRYCP6iJdLrAyv2a0FaSsTYZ5ziWTf3Hno1TQ3NmHP0D, 5785305945... in GRP: 6SsQ/HAHUn...}, " +
		"5: {15ufnw07pVsMwNYUTIiFNYQay+BwmwdYCD9h03W8ArQD, 2010156224... in GRP: 6SsQ/HAHUn...}, " +
		"6: {3RqsBM4ux44bC6+uiBuCp1EQikLtPJA8qkNGWnhiBhYD, 2643318057... in GRP: 6SsQ/HAHUn...}, " +
		"7: {55ai4SlwXic/BckjJoKOKwVuOBdljhBhSYlH/fNEQQ4D, 6482807720... in GRP: 6SsQ/HAHUn...}, " +
		"8: {9PkZKU50joHnnku9b+NM3LqEPujWPoxP/hzr6lRtj6wD, 6603068123... in GRP: 6SsQ/HAHUn...}" +
		"}, " +
		"DhKeys:{" +
		"Grcjbkt1IWKQzyvrQsPKJzKFYPGqwGfOpui/RtSrK0YD: 6342989043... in GRP: 6SsQ/HAHUn..., " +
		"QCxg8d6XgoPUoJo2+WwglBdG4+1NpkaprotPp7T8OiAD: 2579328386... in GRP: 6SsQ/HAHUn..., " +
		"invD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHAD: 1688982497... in GRP: 6SsQ/HAHUn..., " +
		"o54Okp0CSry8sWk5e7c05+8KbgHxhU3rX+Qk/vesIQgD: 5552242738... in GRP: 6SsQ/HAHUn..., " +
		"wRYCP6iJdLrAyv2a0FaSsTYZ5ziWTf3Hno1TQ3NmHP0D: 2812078897... in GRP: 6SsQ/HAHUn..., " +
		"15ufnw07pVsMwNYUTIiFNYQay+BwmwdYCD9h03W8ArQD: 2588260662... in GRP: 6SsQ/HAHUn..., " +
		"3RqsBM4ux44bC6+uiBuCp1EQikLtPJA8qkNGWnhiBhYD: 4967151805... in GRP: 6SsQ/HAHUn..., " +
		"55ai4SlwXic/BckjJoKOKwVuOBdljhBhSYlH/fNEQQ4D: 3187530437... in GRP: 6SsQ/HAHUn..., " +
		"9PkZKU50joHnnku9b+NM3LqEPujWPoxP/hzr6lRtj6wD: 4832738218... in GRP: 6SsQ/HAHUn..." +
		"}}"

	if grp.GoString() != expected {
		t.Errorf("GoString failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, grp.GoString())
	}
}

// Test that Group.GoString returns the expected string for a nil group.
func TestGroup_GoString_NilGroup(t *testing.T) {
	grp := Group{}
	expected := "{" +
		"Name:\"\", " +
		"ID:<nil>, " +
		"Key:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"IdPreimage:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"KeyPreimage:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=, " +
		"InitMessage:\"\", " +
		"Created:0001-01-01 00:00:00 +0000 UTC, " +
		"Members:{<nil>}, " +
		"DhKeys:{}" +
		"}"

	if grp.GoString() != expected {
		t.Errorf("GoString failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, grp.GoString())
	}
}
