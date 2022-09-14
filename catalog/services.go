////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package catalog

import "gitlab.com/elixxir/crypto/sih"

const (
	Default = sih.Default

	//e2e
	Silent = "silent"
	E2e    = "e2e"

	//auth
	Request      = "request"
	Reset        = "reset"
	Confirm      = "confirm"
	ConfirmReset = "confirmReset"

	RequestEphemeral      = "requestEph"
	ResetEphemeral        = "resetEph"
	ConfirmEphemeral      = "confirmEph"
	ConfirmResetEphemeral = "confirmResetEph"

	Group   = "group"
	EndFT   = "endFT"
	GroupRq = "groupRq"

	RestLike = "restLike"
)
