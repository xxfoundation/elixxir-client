package params

import "time"

type CMIX struct {
	//maximum number of rounds to try and send on
	RoundTries uint
	Timeout    time.Duration
	RetryDelay time.Duration
}

func GetDefaultCMIX() CMIX {
	return CMIX{
		RoundTries: 10,
		Timeout:    25 * time.Second,
		RetryDelay: 1 * time.Second,
	}
}
