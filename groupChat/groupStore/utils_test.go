////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupStore

import (
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
	"time"
)

const (
	groupName        = "groupName"
	groupSalt        = "salt"
	groupKey         = "key"
	groupIdPreimage  = "idPreimage"
	groupKeyPreimage = "keyPreimage"
	initMessage      = "initMessage"
)

var created = time.Date(1955, 11, 5, 12, 1, 0, 0, time.Local)

// createTestGroup generates a new group for testing.
func createTestGroup(rng *rand.Rand, t *testing.T) Group {
	members := createMembership(rng, 10, t)
	dkl := GenerateDhKeyList(members[0].ID, randCycInt(rng), members, getGroup())
	return NewGroup(
		[]byte(groupName),
		id.NewIdFromUInt(rng.Uint64(), id.Group, t),
		newKey(groupKey),
		newIdPreimage(groupIdPreimage),
		newKeyPreimage(groupKeyPreimage),
		[]byte(initMessage),
		created,
		members,
		dkl,
	)
}

// createMembership creates a new membership with the specified number of
// randomly generated members.
func createMembership(rng *rand.Rand, size int, t *testing.T) group.Membership {
	contacts := make([]contact.Contact, size)
	for i := range contacts {
		contacts[i] = randContact(rng)
	}

	membership, err := group.NewMembership(contacts[0], contacts[1:]...)
	if err != nil {
		t.Errorf("Failed to create new membership: %+v", err)
	}

	return membership
}

// createDhKeyList creates a new DhKeyList with the specified number of randomly
// generated members.
func createDhKeyList(rng *rand.Rand, size int, _ *testing.T) DhKeyList {
	dkl := make(DhKeyList, size)
	for i := 0; i < size; i++ {
		dkl[*randID(rng, id.User)] = randCycInt(rng)
	}

	return dkl
}

// randMember returns a Member with a random ID and DH public key.
func randMember(rng *rand.Rand) group.Member {
	return group.Member{
		ID:    randID(rng, id.User),
		DhKey: randCycInt(rng),
	}
}

// randContact returns a contact with a random ID and DH public key.
func randContact(rng *rand.Rand) contact.Contact {
	return contact.Contact{
		ID:       randID(rng, id.User),
		DhPubKey: randCycInt(rng),
	}
}

// randID returns a new random ID of the specified type.
func randID(rng *rand.Rand, t id.Type) *id.ID {
	newID, _ := id.NewRandomID(rng, t)
	return newID
}

// randCycInt returns a random cyclic int.
func randCycInt(rng *rand.Rand) *cyclic.Int {
	return getGroup().NewInt(rng.Int63())
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

func newSalt(s string) [group.SaltLen]byte {
	var salt [group.SaltLen]byte
	copy(salt[:], s)
	return salt
}

func newKey(s string) group.Key {
	var key group.Key
	copy(key[:], s)
	return key
}

func newIdPreimage(s string) group.IdPreimage {
	var preimage group.IdPreimage
	copy(preimage[:], s)
	return preimage
}

func newKeyPreimage(s string) group.KeyPreimage {
	var preimage group.KeyPreimage
	copy(preimage[:], s)
	return preimage
}
