////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"testing"
	"time"

	"gitlab.com/elixxir/client/v5/stoppable"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

// TestCritical tests the basic functions of the critical messaging system
func TestCritical(t *testing.T) {
	// Init mock structures & start thread
	kv := versioned.NewKV(ekv.MakeMemstore())
	mr := &mockRoundEventRegistrar{
		statusReturn: true,
	}
	c := newCritical(kv, &mockMonitor{}, mr, mockCriticalSender)
	s := stoppable.NewSingle("test")
	go c.runCriticalMessages(s)

	// Case 1 - should fail
	recipientID := id.NewIdFromString("zezima", id.User, t)
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(500 * time.Millisecond)

	// Case 2 - should succeed
	mr.statusReturn = false
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(500 * time.Millisecond)

	// Case 3 - should fail
	c.send = mockFailCriticalSender
	c.Add(format.NewMessage(2048), recipientID, GetDefaultCMIXParams())
	c.trigger <- true
	time.Sleep(time.Second)
	err := s.Close()
	if err != nil {
		t.Errorf("Failed to stop critical: %+v", err)
	}
}
