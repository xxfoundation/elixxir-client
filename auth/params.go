package auth

import "gitlab.com/elixxir/client/catalog"

type Param struct {
	ReplayRequests bool

	RequestTag      string
	ConfirmTag      string
	ResetRequestTag string
	ResetConfirmTag string
}

func GetDefaultParams() Param {
	return Param{
		ReplayRequests:  false,
		RequestTag:      catalog.Request,
		ConfirmTag:      catalog.Confirm,
		ResetRequestTag: catalog.Reset,
		ResetConfirmTag: catalog.ConfirmReset,
	}
}

func GetDefaultTemporaryParams() Param {
	p := GetDefaultParams()
	p.RequestTag = catalog.RequestEphemeral
	p.ConfirmTag = catalog.ConfirmEphemeral
	p.ResetRequestTag = catalog.ResetEphemeral
	p.ResetConfirmTag = catalog.ConfirmResetEphemeral
	return p
}

func (p Param) getConfirmTag(reset bool) string {
	if reset {
		return p.ResetConfirmTag
	} else {
		return p.ConfirmTag
	}
}
