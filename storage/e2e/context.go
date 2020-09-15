package e2e

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
)

type context struct {
	fa fingerprintAccess

	grp *cyclic.Group

	rng *fastRNG.StreamGenerator
}
