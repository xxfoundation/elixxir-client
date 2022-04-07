package receptionID

import (
	"fmt"
	"gitlab.com/elixxir/client/cmix/identity/receptionID/store"
	"strconv"
	"strings"
)

type IdentityUse struct {
	Identity

	// Denotes if the identity is fake, in which case we do not process messages
	Fake bool

	UR *store.UnknownRounds
	ER *store.EarliestRound
	CR *store.CheckedRounds
}

// GoString returns a string representations of all the values in the
// IdentityUse. This function adheres to the fmt.GoStringer interface.
func (iu IdentityUse) GoString() string {
	str := []string{
		"Identity:" + iu.Identity.GoString(),
		"StartValid:" + iu.StartValid.String(),
		"EndValid:" + iu.EndValid.String(),
		"Fake:" + strconv.FormatBool(iu.Fake),
		"UR:" + fmt.Sprintf("%+v", iu.UR),
		"ER:" + fmt.Sprintf("%+v", iu.ER),
		"CR:" + fmt.Sprintf("%+v", iu.CR),
	}

	return "{" + strings.Join(str, ", ") + "}"
}
