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
