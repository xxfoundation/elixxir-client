package params

import "time"

type Rekey struct {
	RoundTimeout             time.Duration
}

func GetDefaultRekey() Rekey {
	return Rekey{
		RoundTimeout: time.Minute,
	}
}
