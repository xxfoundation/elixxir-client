package params

type Messages struct {
	MessageReceptionBuffLen        uint
	MessageReceptionWorkerPoolSize uint
}

func GetDefaultMessage() Messages {
	return Messages{
		MessageReceptionBuffLen:        500,
		MessageReceptionWorkerPoolSize: 4,
	}
}
