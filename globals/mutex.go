////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import "sync"
// TODO move to transmitMessage.go
var TransmissionMutex = &sync.Mutex{}
