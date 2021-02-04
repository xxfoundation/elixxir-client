///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package contact

import (
	"crypto"
	"encoding/base64"
	"encoding/json"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// Tests marshaling and unmarshalling of a common Contact.
func TestContact_Marshal_Unmarshal(t *testing.T) {
	expectedContact := Contact{
		ID:       id.NewIdFromUInt(rand.Uint64(), id.User, t),
		DhPubKey: getCycInt(256),
		Facts: fact.FactList{
			{Fact: "myUsername", T: fact.Username},
			{Fact: "devinputvalidation@elixxir.io", T: fact.Email},
			{Fact: "6502530000US", T: fact.Phone},
			{Fact: "6502530001US", T: fact.Phone},
		},
	}

	buff := expectedContact.Marshal()

	testContact, err := Unmarshal(buff)
	if err != nil {
		t.Errorf("Unmarshal() produced an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedContact, testContact) {
		t.Errorf("Unmarshaled Contact does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expectedContact, testContact)
	}
}

// Tests marshaling and unmarshalling of a Contact with nil fields.
func TestContact_Marshal_Unmarshal_Nil(t *testing.T) {
	expectedContact := Contact{}

	buff := expectedContact.Marshal()

	testContact, err := Unmarshal(buff)
	if err != nil {
		t.Errorf("Unmarshal() produced an error: %+v", err)
	}

	if !reflect.DeepEqual(expectedContact, testContact) {
		t.Errorf("Unmarshaled Contact does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expectedContact, testContact)
	}
}

// Tests the size of marshaling and JSON marshaling of a Contact with a large
// amount of data.
func TestContact_Marshal_Size(t *testing.T) {
	expectedContact := Contact{
		ID:             id.NewIdFromUInt(rand.Uint64(), id.User, t),
		DhPubKey:       getCycInt(512),
		OwnershipProof: make([]byte, 1024),
		Facts: fact.FactList{
			{Fact: "myVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryLongUsername", T: fact.Username},
			{Fact: "myVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryVeryLongEmail@elixxir.io", T: fact.Email},
			{Fact: "6502530000US", T: fact.Phone},
		},
	}
	rand.Read(expectedContact.OwnershipProof)

	buff := expectedContact.Marshal()

	marshalBuff, err := json.Marshal(expectedContact)
	if err != nil {
		t.Errorf("Marshal() produced an error: %+v", err)
	}

	t.Logf("size of buff:        %d", len(buff))
	t.Logf("size of marshalBuff: %d", len(marshalBuff))
	t.Logf("ratio: %.2f%%", float32(len(buff))/float32(len(marshalBuff))*100)
	t.Logf("%s", marshalBuff)

	if len(marshalBuff) < len(buff) {
		t.Errorf("JSON Contact smaller than marshaled contact."+
			"\nJSON:    %d\nmarshal: %d", len(marshalBuff), len(buff))
	}
}

// Unit test of getFingerprint
func TestContact_GetFingerprint(t *testing.T) {
	c := Contact{
		ID:       id.NewIdFromString("Samwise", id.User, t),
		DhPubKey: getCycInt(512),
	}

	contactString := c.GetFingerprint()
	if len(contactString) != fingerprintLength {
		t.Errorf("Unexpected length for fingerprint."+
			"\n\tExpected length: %d"+
			"\n\tReceived length: %d", len(contactString), fingerprintLength)
	}

	// Generate hash
	sha := crypto.SHA256
	h := sha.New()

	// Hash Id and public key
	h.Write(c.ID.Bytes())
	h.Write(c.DhPubKey.Bytes())
	data := h.Sum(nil)

	expectedFP := base64.StdEncoding.EncodeToString(data[:])[:fingerprintLength]

	if strings.Compare(contactString, expectedFP) != 0 {
		t.Errorf("Fingerprint outputted is not expected."+
			"\n\tExpected: %s"+
			"\n\tReceived: %s", contactString, expectedFP)
	}

}

// Consistency test for changes in underlying dependencies
func TestContact_GetFingerprint_Consistency(t *testing.T) {
	expected := []string{
		"rBUw1n4jtH4uEYq",
		"Z/Jm1OUwDaql5cd",
		"+vHLzY+yH96zAiy",
		"cZm5Iz78ViOIlnh",
		"9LqrcbFEIV4C4LX",
		"ll4eykGpMWYlxw+",
		"6YQshWJhdPL6ajx",
		"Y6gTPVEzow4IHOm",
		"6f/rT2vWxDC9tdt",
		"rwqbDT+PoeA6Iww",
		"YN4IFijP/GZ172O",
		"ScbHVQc2T9SXQ2m",
		"50mfbCXQ+LIqiZn",
		"cyRYdMKXByiFdtC",
		"7g6ujy7iIbJVl4F",
	}

	numTest := 15
	output := make([]string, 0)
	for i := 0; i < numTest; i++ {
		c := Contact{
			ID:       id.NewIdFromUInt(uint64(i), id.User, t),
			DhPubKey: getGroup().NewInt(25),
		}

		contactString := c.GetFingerprint()
		output = append(output, contactString)
	}

	for i := 0; i < numTest; i++ {
		if strings.Compare(output[i], expected[i]) != 0 {
			t.Errorf("Fingerprint outputted is not expected."+
				"\n\tReceived: %s"+
				"\n\tExpected: %s", output[i], expected[i])
		}

	}

}

func getCycInt(size int) *cyclic.Int {
	var primeString = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
		"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
		"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
		"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
		"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
		"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
		"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
		"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
		"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
		"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
		"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
		"FFFFFFFFFFFFFFFF"
	buff, err := csprng.GenerateInGroup([]byte(primeString), size, csprng.NewSystemRNG())
	if err != nil {
		panic(err)
	}

	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16), large.NewInt(2)).NewIntFromBytes(buff)
	return grp
}

func getGroup() *cyclic.Group {
	return cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D4941"+
			"3394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688"+
			"B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861"+
			"575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC"+
			"718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FF"+
			"B1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBC"+
			"A23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD"+
			"161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2B"+
			"EEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C"+
			"4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F"+
			"1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F"+
			"96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA7554715"+
			"84A09EC723742DC35873847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16))
}
