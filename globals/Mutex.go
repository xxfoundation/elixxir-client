package globals

import "sync"

var TransmissionMutex *sync.Mutex = &sync.Mutex{}
