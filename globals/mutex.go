////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"sync"
	"time"
)

// TODO move to transmitMessage.go
const DefaultTransmitDelay = 1000

var BlockingTransmission = true
var TransmissionMutex = &sync.Mutex{}
var TransmissionErrCh = make(chan error, 100)
var TransmitDelay = time.Duration(DefaultTransmitDelay) * time.Millisecond
