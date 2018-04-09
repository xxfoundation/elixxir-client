////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import "sync"

// TODO move to transmitMessage.go
var BlockingTransmission = true
var TransmissionMutex = &sync.Mutex{}
var TransmissionErrCh = make(chan error, 100)
