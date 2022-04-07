package message

import "time"

type Params struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksInProcessMessage      uint
	InProcessMessageWait           time.Duration
	RealtimeOnly                   bool
}

func GetDefaultParams() Params {
	return Params{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
		MaxChecksInProcessMessage:      10,
		InProcessMessageWait:           15 * time.Minute,
		RealtimeOnly:                   false,
	}
}
