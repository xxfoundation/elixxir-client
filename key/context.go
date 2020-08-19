package key

import (
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/cyclic"
)

type context struct {
	fa fingerprintAccess

	grp *cyclic.Group

	kv *storage.Session
}
