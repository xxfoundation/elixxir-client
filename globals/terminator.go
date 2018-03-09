////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import "time"

type ThreadTerminator chan chan bool

func NewThreadTerminator() ThreadTerminator {
	t := make(chan chan bool)
	return t
}

func (t ThreadTerminator) Terminate() {
	t <- nil
}

// Try's to kill a thread controlled by a termination channel for the length of
// the timeout, returns its success. pass 0 for no timeout
func (t ThreadTerminator) BlockingTerminate(timeout uint64) bool {

	killNotify := make(chan bool)
	defer close(killNotify)

	if timeout != 0 {
		timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
		defer timer.Stop()

		t <- killNotify

		select {
		case _ = <-killNotify:
			return true
		case <-timer.C:
			return false
		}
	} else {
		_ = <-killNotify
		return true
	}
}
