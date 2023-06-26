////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receptionID

import (
	"fmt"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID/store"
	"strconv"
	"strings"
)

type IdentityUse struct {
	Identity `json:""`

	// Denotes if the identity is fake, in which case we do not process messages
	Fake bool `json:"fake"`

	UR *store.UnknownRounds `json:"ur"`
	ER *store.EarliestRound `json:"er"`
	CR *store.CheckedRounds `json:"cr"`
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
