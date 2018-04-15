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
const DEFAULT_TRANSMIT_DELAY = 1000

var BlockingTransmission = true
var TransmissionMutex = &sync.Mutex{}
var TransmissionErrCh = make(chan error, 100)
var TransmitDelay = time.Duration(DEFAULT_TRANSMIT_DELAY) * time.Millisecond
var ReceptionCounter = uint64(0)
