////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"reflect"
	"testing"
	"time"
)

// Tests that listenerTracker.Hear correctly updates the last use timestamp and
// calls the wrapped Hear function.
func Test_listenerTracker_Hear(t *testing.T) {
	lastUsed, closed := int64(0), uint32(0)
	itemChan := make(chan receive.Message, 1)
	lt := &listenerTracker{
		h: &handler{
			lastUse: &lastUsed,
			closed:  &closed,
		},
		l: &mockListener{itemChan, "mockListener"},
	}

	expected := receive.Message{
		Payload:     []byte("Message payload."),
		Sender:      id.NewIdFromString("senderID", id.User, t),
		RecipientID: id.NewIdFromString("RecipientID", id.User, t),
	}

	lt.Hear(expected)

	select {
	case r := <-itemChan:
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("Did not receive expected receive.Message."+
				"\nexpected: %+v\nreceived: %+v", expected, r)
		}
		if netTime.Since(lt.h.LastUse()) > 300*time.Millisecond {
			t.Errorf("Last use has incorrect time: %s", lt.h.LastUse())
		}

	case <-time.After(30 * time.Millisecond):
		t.Error("Timed out waiting for Hear to be called.")
	}
}

// Tests that listenerTracker.Name calls the wrapped listener Name function.
func Test_listenerTracker_Name(t *testing.T) {
	expected := "mockListener"
	lt := &listenerTracker{
		l: &mockListener{make(chan receive.Message, 1), "mockListener"},
	}

	if lt.Name() != expected {
		t.Errorf("Did not get expected name.\nexected: %s\nreceived: %s",
			expected, lt.Name())
	}
}

type mockListener struct {
	item chan receive.Message
	name string
}

func (m *mockListener) Hear(item receive.Message) { m.item <- item }
func (m *mockListener) Name() string              { return m.name }
