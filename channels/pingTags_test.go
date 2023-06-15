////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

// Tests that all combinations of defined PingType passed into
// getRankingPingType result in the expected type.
func Test_getRankingPingType(t *testing.T) {
	tests := []struct{ a, b, expected PingType }{
		{GenericPing, ReplyPing, ReplyPing},
		{GenericPing, MentionPing, MentionPing},
		{ReplyPing, GenericPing, ReplyPing},
		{ReplyPing, MentionPing, ReplyPing},
		{MentionPing, GenericPing, MentionPing},
		{MentionPing, ReplyPing, ReplyPing},
	}

	for i, tt := range tests {
		pt := getRankingPingType(tt.a, tt.b)
		if tt.expected != pt {
			t.Errorf("Unexpected PingType when comparing %q with %q (%d)."+
				"\nexpected: %q\nreceived: %q", tt.a, tt.b, i, tt.expected, pt)
		}
	}
}

// Unit test of makeUserPingTags.
func Test_makeUserPingTags(t *testing.T) {
	prng := rand.New(rand.NewSource(579498))
	types := []PingType{ReplyPing, MentionPing}

	for i := 0; i < 6; i++ {
		pings := map[PingType][]ed25519.PublicKey{
			ReplyPing: {}, MentionPing: {},
		}
		expected := make([]string, i)
		for j := 0; j < i; j++ {
			pt := types[prng.Intn(len(types))]
			user, _, _ := ed25519.GenerateKey(prng)
			pings[pt] = append(pings[pt], user)
			expected[j] = makeUserPingTag(user, pt)
		}
		if i == 0 {
			expected = nil
		}
		tags := makeUserPingTags(pings)
		sort.Strings(expected)
		sort.Strings(tags)
		if !reflect.DeepEqual(expected, tags) {
			t.Errorf("Unexpected tags (%d).\nexpected: %#v\nreceived: %#v",
				i, expected, tags)
		}
	}
}

// Consistency test of makeUserPingTag.
func TestManager_makeUserPingTag_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(579498))
	expected := []string{
		"32b15a6bd6db85bf239a7fe6aeb84072c2646b96188c9c4f7dec6d35928436c2-usrReply",
		"88ab667cdc60e041849a924ed7b6e60fa2de62eca4b67336c86e7620a19d270f-usrReply",
		"ddab884d040d9182950aca0eab2899977caf62eef098d0973f99aa1b4c97f541-usrMention",
		"76341ec7e5da6fe34b80021c4e0f6362d3a1a37091126f8cb74ec8ee5bb375fb-usrMention",
		"d0f05d30b6d62460788ac7a7bde3b0de8065c6b1673ba3a724af90e87f1a152e-usrMention",
		"69a8a814d254be6d3652c5f013d063803c6542a94c0dcd52cb9d9f68bcf14041-usrMention",
		"215a17025a25d892baa595d6677046fe74d625505ad65a64d367975c94477831-usrMention",
		"9b6400094a53921ff041c024265748df635843cb6a50e8a4d6f68f7c52f9e766-usrMention",
		"456e99e886cf889b6b9ce63ecece752267b199032a2236239aaedf9d519a491b-usrReply",
		"2cf8839b549c5b360fb5434de944918d5680e203e173889fd48dd4c91099762e-usrMention",
		"a0a48cffe820a8f7e213b6bb99b99c1270fc8126bbde165b756449895f6f6603-usrReply",
		"d1086f945874607d89d9cc59c6e940217bd69c8d71dc45a0f3728cb6146ae154-usrReply",
		"db60467fc127405470c73cbc4734d1098d43e85c61904a114d95466cfb30781c-usrReply",
		"7bd7b36cf1393fdf8576df1182e5b668bb3f021aa4a1b562d885711153a30395-usrReply",
		"61a25b1fbb5dd9490d43b64d6899ae3f1fac2c2e7bf8cade5fbbcced897f10fe-usrReply",
		"689c8cc76cb15c457f6666457b360701d4f1a6312f6f1a1f1c910353a5d0ce10-usrMention",
		"64c7b1b343f38077d30cb9b99a0ed2844974d984fbd9630999a4a59069a964a7-usrReply",
		"f56ee72f86ee39d52887a08927606bce8a9d21726a82792655aee098705c48a9-usrReply",
		"008da28a9f8f70dafe533283acabbed77cb92bf5058f044bb50e1167ce6f05b4-usrMention",
		"70a659261ad311cf6111639ab6be9677f5afc0ab9e13546b3afc64597ccaabfd-usrReply",
	}
	types := []PingType{ReplyPing, MentionPing}

	for i, exp := range expected {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		tag := makeUserPingTag(pubKey, types[prng.Intn(len(types))])
		if exp != tag {
			t.Errorf("Unexpected tag for key %X (%d)."+
				"\nexpected: %s\nreceived: %s", pubKey, i, exp, tag)
		}
	}
}

// Unit test of pingTypeFromTag.
func Test_pingTypeFromTag(t *testing.T) {
	prng := rand.New(rand.NewSource(65881))
	types := []PingType{ReplyPing, MentionPing}

	for i := 0; i < 10; i++ {
		pubKey, _, _ := ed25519.GenerateKey(prng)
		expected := types[prng.Intn(len(types))]
		tag := makeUserPingTag(pubKey, expected)

		pt, err := pingTypeFromTag(tag)
		if err != nil {
			t.Errorf("Failed to get type (%d): %+v", i, err)
		}
		if expected != pt {
			t.Errorf("Unexpected PingType (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, pt)
		}
	}
}

// Error path: Tests that pingTypeFromTag returns an error for an invalid tag.
func Test_typeFromPingTag_InvalidTagError(t *testing.T) {
	_, err := pingTypeFromTag(
		"64c7b1b343f38077d30cb9b99a0ed2844974d984fbd9630999a4a59069a964a7")
	if err == nil {
		t.Errorf("Did not get error for invalid tag.")
	}
}
