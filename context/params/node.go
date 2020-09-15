package params

//import (
//	"time"
//)

type NodeKeys struct {
	WorkerPoolSize uint
}

func GetDefaultNodeKeys() NodeKeys {
	return NodeKeys{
		WorkerPoolSize: 10,
	}
}
