////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Unit test of GenerateDhKeyList.
func TestGenerateDhKeyList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	grp := getGroup()
	userID := id.NewIdFromString("userID", id.User, t)
	privKey := grp.NewInt(42)
	members := createMembership(prng, 10, t)

	// Set one of the members as the sender (the current user)
	senderIndex := 2
	members[senderIndex].ID = userID
	members[senderIndex].DhKey = grp.ExpG(privKey, grp.NewInt(1))

	dkl := GenerateDhKeyList(userID, privKey, members, grp)

	for i, m := range members {
		dhKey, exists := dkl[*m.ID]
		if i == senderIndex {
			// Make sure the sender is not in the list
			if exists {
				t.Errorf("Found DH key for sender (member #%d with ID %s) in "+
					"DH key list.", i, m.ID)
			}
			continue
		} else if !exists {
			// Ensure a DH key exists in the list for this member
			t.Errorf("No DH key for member #%d with ID %s in DH key list.",
				i, m.ID)
		}

		// Make sure that the DH key is correct
		if dhKey.Cmp(m.DhKey) == -2 {
			t.Errorf("DH key in list for member #%d with ID %s incorrect."+
				"\nexpected: %s\nreceived: %s",
				i, m.ID, m.DhKey.Text(10), dhKey.Text(10))
		}
	}
}

// Unit test of DhKeyList.DeepCopy.
func TestDhKeyList_DeepCopy(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	dkl := createDhKeyList(prng, 10, t)
	newDkl := dkl.DeepCopy()

	if !reflect.DeepEqual(dkl, newDkl) {
		t.Errorf("DeepCopy() failed to return a copy of the original."+
			"\nexpected: %#v\nrecevied: %#v", dkl, newDkl)
	}

	if &dkl == &newDkl {
		t.Errorf("DeepCopy returned a copy of the pointer."+
			"\nexpected: %p\nreceived: %p", &dkl, &newDkl)
	}
}

// Tests that a DhKeyList that is serialized and deserialized matches the
// original.
func TestDhKeyList_Serialize_DeserializeDhKeyList(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	dkl := createDhKeyList(prng, 10, t)

	data := dkl.Serialize()
	newDkl, err := DeserializeDhKeyList(data)
	if err != nil {
		t.Errorf("DeserializeDhKeyList returned an error: %+v", err)
	}

	if !reflect.DeepEqual(dkl, newDkl) {
		t.Errorf("Failed to serialize and deserialize DhKeyList."+
			"\nexpected: %#v\nreceived: %#v", dkl, newDkl)
	}
}

// Error path: an error is returned when DeserializeDhKeyList encounters invalid
// cyclic int.
func TestDeserializeDhKeyList_DhKeyBinaryDecodeError(t *testing.T) {
	expectedErr := strings.SplitN(dhKeyDecodeErr, "%", 2)[0]

	_, err := DeserializeDhKeyList(make([]byte, 41))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("DeserializeDhKeyList failed to return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Unit test of DhKeyList.GoString.
func TestDhKeyList_GoString(t *testing.T) {
	grp := createTestGroup(rand.New(rand.NewSource(42)), t)
	expected := "{Grcjbkt1IWKQzyvrQsPKJzKFYPGqwGfOpui/RtSrK0YD: 6342989043..." +
		" in GRP: 6SsQ/HAHUn..., QCxg8d6XgoPUoJo2+WwglBdG4+1NpkaprotPp7T8OiAD" +
		": 2579328386... in GRP: 6SsQ/HAHUn..., invD4ElbVxL+/b4MECiH4QDazS2IX" +
		"2kstgfaAKEcHHAD: 1688982497... in GRP: 6SsQ/HAHUn..., o54Okp0CSry8sW" +
		"k5e7c05+8KbgHxhU3rX+Qk/vesIQgD: 5552242738... in GRP: 6SsQ/HAHUn...," +
		" wRYCP6iJdLrAyv2a0FaSsTYZ5ziWTf3Hno1TQ3NmHP0D: 2812078897... in GRP:" +
		" 6SsQ/HAHUn..., 15ufnw07pVsMwNYUTIiFNYQay+BwmwdYCD9h03W8ArQD: 258826" +
		"0662... in GRP: 6SsQ/HAHUn..., 3RqsBM4ux44bC6+uiBuCp1EQikLtPJA8qkNGW" +
		"nhiBhYD: 4967151805... in GRP: 6SsQ/HAHUn..., 55ai4SlwXic/BckjJoKOKw" +
		"VuOBdljhBhSYlH/fNEQQ4D: 3187530437... in GRP: 6SsQ/HAHUn..., 9PkZKU5" +
		"0joHnnku9b+NM3LqEPujWPoxP/hzr6lRtj6wD: 4832738218... in GRP: 6SsQ/HA" +
		"HUn...}"
	if grp.DhKeys.GoString() != expected {
		t.Errorf("GoString failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, grp.DhKeys.GoString())
	}
}

// Tests that DhKeyList.GoString. returns the expected string for a nil map.
func TestDhKeyList_GoString_NilMap(t *testing.T) {
	dkl := DhKeyList{}
	expected := "{}"

	if dkl.GoString() != expected {
		t.Errorf("GoString failed to return the expected string."+
			"\nexpected: %s\nreceived: %s", expected, dkl.GoString())
	}
}
