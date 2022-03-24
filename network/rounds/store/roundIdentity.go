package store

import (
	"encoding/base64"
	"encoding/binary"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// roundIdentitySize is the size of a roundIdentity.
const roundIdentitySize = 32

// roundIdentity uniquely identifies a round ID for a specific identity.
type roundIdentity [roundIdentitySize]byte

// newRoundIdentity generates a new unique round identifier for the round ID,
// recipient ID, and address ID.
func newRoundIdentity(rid id.Round, recipient *id.ID, ephID ephemeral.Id) roundIdentity {
	h, _ := hash.NewCMixHash()
	ridBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(ridBytes, uint64(rid))
	h.Write(ridBytes)
	h.Write(recipient[:])
	h.Write(ephID[:])
	riBytes := h.Sum(nil)

	ri := unmarshalRoundIdentity(riBytes)

	return ri
}

// String prints a base 64 string representation of roundIdentity. This function
// satisfies the fmt.Stringer interface.
func (ri roundIdentity) String() string {
	return base64.StdEncoding.EncodeToString(ri[:])
}

// Marshal returns the roundIdentity as a byte slice.
func (ri roundIdentity) Marshal() []byte {
	return ri[:]
}

// unmarshalRoundIdentity unmarshalls the byte slice into a roundIdentity.
func unmarshalRoundIdentity(b []byte) roundIdentity {
	var ri roundIdentity
	copy(ri[:], b)

	return ri
}
