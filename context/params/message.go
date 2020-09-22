package params

import "time"

type Messages struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
	MaxChecksGarbledMessage        uint
	GarbledMessageWait             time.Duration
}

func GetDefaultMessage() Messages {
	return Messages{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
		MaxChecksGarbledMessage:        10,
		GarbledMessageWait:             15 * time.Minute,
	}
}
