////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"testing"
	"time"
)

func TestNewThreadTerminator(t *testing.T) {

	term := NewThreadTerminator()

	var success bool

	go func(term ThreadTerminator) {
		term <- nil
	}(term)

	timer := time.NewTimer(time.Duration(1000) * time.Millisecond)
	defer timer.Stop()

	select {
	case _ = <-term:
		success = true
	case <-timer.C:
		success = false
	}

	if !success {
		t.Errorf("NewThreadTerminator: Could not use the ThreadTerminator to" +
			" stop a thread")
	}

}

func TestBlockingTerminate(t *testing.T) {

	term := NewThreadTerminator()

	go func(term ThreadTerminator) {
		var killNotify chan<- bool

		q := false

		for !q {
			select {
			case killNotify = <-term:
				q = true
			}

			close(term)

			killNotify <- true

		}
	}(term)

	success := term.BlockingTerminate(1000)

	if !success {
		t.Errorf("BlockingTerminate: Thread did not terminate in time")
	}

}
