////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"encoding/json"
	"github.com/cloudflare/circl/dh/sidh"
	util "gitlab.com/elixxir/client/v4/storage/utility"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"math/rand"
	"reflect"
	"testing"
)

func TestGetSessionIDFromBaseKey(t *testing.T) {
	rng := rand.New(rand.NewSource(38088973))
	grp := getGroup()

	expected := []SessionID{
		{46, 18, 71, 24, 66, 93, 105, 29, 204, 46, 204, 12, 252, 77, 6, 55, 0, 188, 252, 25, 75, 86, 153, 133, 143, 227, 101, 3, 113, 234, 175, 96},
		{185, 179, 205, 70, 150, 238, 102, 217, 53, 70, 73, 16, 5, 238, 67, 186, 98, 152, 149, 32, 58, 14, 179, 83, 67, 192, 105, 171, 251, 196, 173, 11},
		{158, 86, 255, 223, 136, 139, 64, 159, 84, 46, 42, 176, 214, 129, 163, 180, 203, 100, 247, 151, 239, 134, 225, 148, 170, 30, 105, 94, 8, 243, 129, 83},
		{235, 253, 193, 91, 32, 78, 5, 109, 138, 213, 170, 199, 149, 44, 189, 223, 126, 253, 176, 83, 66, 137, 102, 21, 0, 170, 220, 227, 128, 89, 222, 51},
		{242, 251, 217, 158, 171, 181, 55, 14, 192, 111, 163, 119, 22, 131, 133, 60, 253, 53, 227, 112, 61, 215, 106, 72, 244, 23, 2, 172, 131, 167, 80, 110},
		{45, 40, 71, 163, 114, 76, 190, 144, 149, 84, 116, 164, 253, 49, 225, 61, 233, 201, 242, 181, 71, 194, 103, 58, 111, 105, 161, 64, 94, 167, 61, 40},
		{64, 136, 240, 139, 1, 242, 207, 34, 75, 84, 225, 28, 12, 126, 121, 204, 230, 9, 192, 124, 255, 109, 25, 184, 40, 111, 185, 133, 97, 100, 128, 245},
		{182, 66, 18, 186, 139, 221, 13, 89, 182, 176, 88, 79, 215, 9, 159, 118, 163, 110, 199, 152, 183, 227, 219, 149, 29, 232, 231, 35, 207, 143, 15, 150},
		{142, 221, 214, 136, 222, 255, 190, 206, 156, 128, 84, 242, 115, 140, 211, 169, 226, 236, 211, 251, 24, 205, 225, 176, 53, 249, 32, 197, 106, 115, 238, 165},
		{201, 55, 113, 120, 204, 81, 35, 70, 33, 160, 218, 154, 195, 213, 43, 61, 55, 140, 7, 246, 128, 174, 5, 140, 254, 174, 154, 10, 78, 27, 124, 8},
	}

	for i, exp := range expected {
		partnerPubKey := dh.GeneratePublicKey(dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng), grp)
		myPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)

		partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
		partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
		_ = partnerSIDHPrivKey.Generate(rng)
		partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
		mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
		mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
		_ = mySIDHPrivKey.Generate(rng)
		mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

		baseKey := GenerateE2ESessionBaseKey(myPrivKey, partnerPubKey, grp,
			mySIDHPrivKey, partnerSIDHPubKey)

		sid := GetSessionIDFromBaseKey(baseKey)

		if sid != exp {
			t.Errorf("Unexpected SessionID (%d).\nexpected: %+v\nreceived: %+v",
				i, exp, sid)
		}
	}
}

// Tests that a SessionID can be marshalled and unmarshalled.
func TestSessionID_Marshal_Unmarshal(t *testing.T) {
	s, _ := makeTestSession()

	sid := GetSessionIDFromBaseKey(s.baseKey)

	data := sid.Marshal()

	var newSID SessionID
	if err := newSID.Unmarshal(data); err != nil {
		t.Errorf("Failed to unmarshal %T: %+v", newSID, err)
	}

	if !reflect.DeepEqual(sid, newSID) {
		t.Errorf("Unexpected unmarshalled session ID."+
			"\nexpected: %#v\nreceived: %#v", sid, newSID)
	}
}

// Tests that a SessionID can be JSON marshalled and unmarshalled.
func TestSessionID_JSON_Marshal_Unmarshal(t *testing.T) {
	s, _ := makeTestSession()

	sid := GetSessionIDFromBaseKey(s.baseKey)

	data, err := json.Marshal(sid)
	if err != nil {
		t.Errorf("Failed to JSON marshal %T: %+v", sid, err)
	}

	var newSID SessionID
	if err = json.Unmarshal(data, &newSID); err != nil {
		t.Errorf("Failed to JSON unmarshal %T: %+v", newSID, err)
	}

	if !reflect.DeepEqual(sid, newSID) {
		t.Errorf("Unexpected unmarshalled session ID."+
			"\nexpected: %#v\nreceived: %#v", sid, newSID)
	}
}
