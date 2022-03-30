package rekey

import "time"

type Params struct {
	RoundTimeout time.Duration
}

func GetDefaultParams() Params {
	return Params{
		RoundTimeout: time.Minute,
	}
}
