////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
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

//Timeout path
func TestBlockingTerminate_Timeout(t *testing.T) {
	term := NewThreadTerminator()

	go func(term ThreadTerminator) {

		q := false
		//Sleep long enough for the blocking terminate to complete below
		time.Sleep(5 * time.Millisecond)
		for !q {
			select {
			case _ = <-term:
				q = true
			}

			close(term)

		}
	}(term)

	//Should return false, as the go func does not complete before the timeout
	fail := term.BlockingTerminate(1)

	if fail {
		t.Errorf("BlockingTerminate: Expected error path, should have timed out")
	}
}

func TestBlockingTerminate_ZeroTimeout(t *testing.T) {
	term := NewThreadTerminator()

	success := term.BlockingTerminate(0)

	if !success {
		t.Errorf("BlockingTerminate: Thread did not terminate in time")
	}
}
