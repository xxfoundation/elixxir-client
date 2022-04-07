package auth

type Param struct {
	ReplayRequests bool

	RequestTag      string
	ConfirmTag      string
	ResetRequestTag string
	ResetConfirmTag string
}

func (p Param) getConfirmTag(reset bool) string {
	if reset {
		return p.ResetConfirmTag
	} else {
		return p.ConfirmTag
	}
}
