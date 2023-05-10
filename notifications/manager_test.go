package notifications

import (
	"gitlab.com/elixxir/ekv"
)

type TestCollectiveKeystore struct {
	memstore ekv.KeyValue
}
