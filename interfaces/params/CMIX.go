package params

import "time"

type CMIX struct {
	//maximum number of rounds to try and send on
	RoundTries uint
	Timeout    time.Duration
}

func GetDefaultCMIX() CMIX {
	return CMIX{
		RoundTries: 3,
		Timeout:    10 * time.Second,
	}
}
