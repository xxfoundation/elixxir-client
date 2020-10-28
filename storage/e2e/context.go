package e2e

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"
)

type context struct {
	fa fingerprintAccess

	grp *cyclic.Group

	rng *fastRNG.StreamGenerator

	myID *id.ID
}
