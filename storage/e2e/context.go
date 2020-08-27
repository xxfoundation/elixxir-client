package e2e

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
)

type context struct {
	fa fingerprintAccess

	grp *cyclic.Group

	kv *versioned.KV
}
