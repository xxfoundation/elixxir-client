package partner

import (
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"testing"
)

// ManagerIdentity.GetMe unit test
func TestManagerIdentity_GetMe(t *testing.T) {
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	myId := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	mid := MakeManagerIdentity(partnerID, myId)

	if !myId.Cmp(mid.GetMe()) {
		t.Fatalf("GetMe did not retrieve expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", myId, mid.GetMe())
	}

}

// ManagerIdentity.GetPartner unit test
func TestManagerIdentity_GetPartner(t *testing.T) {
	partnerID := id.NewIdFromUInt(rand.Uint64(), id.User, t)
	myId := id.NewIdFromUInt(rand.Uint64(), id.User, t)

	mid := MakeManagerIdentity(partnerID, myId)

	if !partnerID.Cmp(mid.GetPartner()) {
		t.Fatalf("GetPartner did not retrieve expected data."+
			"\nExpected: %v"+
			"\nReceived: %v", partnerID, mid.GetPartner())
	}

}
