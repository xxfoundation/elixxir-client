////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receive

import (
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"testing"
	"time"
)

// tests that New create a correctly structured switchboard
func TestNew(t *testing.T) {
	sw := New()

	if sw.id == nil {
		t.Errorf("did not create an id map")
	}

	if sw.messageType == nil {
		t.Errorf("did not create a messageType map")
	}
}

//Tests that register listener handles errors properly
func TestSwitchboard_RegisterListener_Error_NilUserID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register listener to nil user") {
			t.Errorf("A nil userID caused the wrong error: %s", r)
		}
	}()

	sw := New()
	sw.RegisterListener(nil, 0, &funcListener{})

	t.Errorf("A nil userID should have caused an panic")
}

//Tests that register listener handles errors properly
func TestSwitchboard_RegisterListener_Error_NilListener(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register nil listener") {
			t.Errorf("A nil listener caused the wrong error: %s", r)
		}
	}()

	sw := New()
	sw.RegisterListener(id.NewIdFromUInt(42, id.User, t), 0, nil)

	t.Errorf("A nil listener should have caused an error")
}

//Tests that RegisterListener properly registers the listeners
func TestSwitchboard_RegisterListener(t *testing.T) {
	sw := New()

	l := &funcListener{}

	uid := id.NewIdFromUInt(42, id.User, t)

	mt := catalog.MessageType(2)

	lid := sw.RegisterListener(uid, mt, l)

	if lid.messageType != mt {
		t.Errorf("ListenerID message type is wrong")
	}

	if !lid.userID.Cmp(uid) {
		t.Errorf("ListenerID userID is wrong")
	}

	if lid.listener != l {
		t.Errorf("ListenerID listener is wrong")
	}

	//check that the listener is registered in the appropriate location
	setID := sw.id.Get(uid)

	if !setID.Has(lid) {
		t.Errorf("Listener is not registered by ID")
	}

	setType := sw.messageType.Get(mt)

	if !setType.Has(lid) {
		t.Errorf("Listener is not registered by Message Tag")
	}

}

//Tests that register funcListener handles errors properly
func TestSwitchboard_RegisterFunc_Error_NilUserID(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register listener to nil user") {
			t.Errorf("A nil userID caused the wrong error: %s", r)
		}
	}()

	sw := New()
	sw.RegisterFunc("test", nil, 0, func(item Message) {})

	t.Errorf("A nil user ID should have caused an error")
}

//Tests that register funcListener handles errors properly
func TestSwitchboard_RegisterFunc_Error_NilFunc(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register function listener 'test' with nil func") {
			t.Errorf("A nil func caused the wrong error: %s", r)
		}
	}()

	sw := New()
	sw.RegisterFunc("test", id.NewIdFromUInt(42, id.User, t), 0, nil)

	t.Errorf("A nil listener func should have caused an error")
}

//Tests that RegisterFunc properly registers the listeners
func TestSwitchboard_RegisterFunc(t *testing.T) {
	sw := New()

	heard := false

	l := func(item Message) { heard = true }

	uid := id.NewIdFromUInt(42, id.User, t)

	mt := catalog.MessageType(1)

	lid := sw.RegisterFunc("test", uid, mt, l)

	if lid.messageType != mt {
		t.Errorf("ListenerID message type is wrong")
	}

	if !lid.userID.Cmp(uid) {
		t.Errorf("ListenerID userID is wrong")
	}

	//check that the listener is registered in the appropriate location
	setID := sw.id.Get(uid)

	if !setID.Has(lid) {
		t.Errorf("Listener is not registered by ID")
	}

	setType := sw.messageType.Get(mt)

	if !setType.Has(lid) {
		t.Errorf("Listener is not registered by Message Tag")
	}

	lid.listener.Hear(Message{})
	if !heard {
		t.Errorf("Func listener not registered correctly")
	}
}

//Tests that register chanListener handles errors properly
func TestSwitchboard_RegisterChan_Error_NilUser(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register listener to nil user") {
			t.Errorf("A nil user ID caused the wrong error: %s", r)
		}
	}()
	sw := New()
	sw.RegisterChannel("test", nil, 0,
		make(chan Message))

	t.Errorf("A nil userID should have caused an error")
}

//Tests that register chanListener handles errors properly
func TestSwitchboard_RegisterChan_Error_NilChan(t *testing.T) {
	defer func() {
		if r := recover(); r != nil && !strings.Contains(r.(string),
			"cannot register channel listener 'test' with nil channel") {
			t.Errorf("A nil channel caused the wrong error: %s", r)
		}
	}()
	sw := New()
	sw.RegisterChannel("test", &id.ID{}, 0, nil)

	t.Errorf("A nil channel func should have caused an error")
}

//Tests that RegisterChan properly registers the listeners
func TestSwitchboard_RegisterChan(t *testing.T) {
	sw := New()

	ch := make(chan Message, 1)

	uid := id.NewIdFromUInt(42, id.User, t)

	mt := catalog.MessageType(1)

	lid := sw.RegisterChannel("test", uid, mt, ch)

	//check the returns
	if lid.messageType != mt {
		t.Errorf("ListenerID message type is wrong")
	}

	if !lid.userID.Cmp(uid) {
		t.Errorf("ListenerID userID is wrong")
	}

	//check that the listener is registered in the appropriate location
	setID := sw.id.Get(uid)

	if !setID.Has(lid) {
		t.Errorf("Listener is not registered by ID")
	}

	setType := sw.messageType.Get(mt)

	if !setType.Has(lid) {
		t.Errorf("Listener is not registered by Message Tag")
	}

	lid.listener.Hear(Message{})
	select {
	case <-ch:
	case <-time.After(5 * time.Millisecond):
		t.Errorf("Chan listener not registered correctly")
	}
}

//tests all combinations of hits and misses for speak
func TestSwitchboard_Speak(t *testing.T) {

	uids := []*id.ID{{}, AnyUser(), id.NewIdFromUInt(42, id.User, t), id.NewIdFromUInt(69, id.User, t)}
	mts := []catalog.MessageType{AnyType, catalog.NoType, catalog.XxMessage}

	for _, uidReg := range uids {
		for _, mtReg := range mts {

			//create the registrations
			sw := New()
			ch1 := make(chan Message, 1)
			ch2 := make(chan Message, 1)

			sw.RegisterChannel("test", uidReg, mtReg, ch1)
			sw.RegisterChannel("test", uidReg, mtReg, ch2)

			//send every possible message
			for _, uid := range uids {
				for _, mt := range mts {
					if uid.Cmp(&id.ID{}) || mt == AnyType {
						continue
					}

					m := Message{
						Payload:     []byte{0, 1, 2, 3},
						Sender:      uid,
						MessageType: mt,
					}

					sw.Speak(m)

					shouldHear := (m.Sender.Cmp(uidReg) ||
						uidReg.Cmp(&id.ID{}) || uidReg.Cmp(AnyUser())) &&
						(m.MessageType == mtReg || mtReg == AnyType)

					var heard1 bool

					select {
					case <-ch1:
						heard1 = true
					case <-time.After(5 * time.Millisecond):
						heard1 = false
					}

					if shouldHear != heard1 {
						t.Errorf("Correct operation not recorded "+
							"for listener 1: Expected: %v, Occured: %v",
							shouldHear, heard1)
					}

					var heard2 bool

					select {
					case <-ch2:
						heard2 = true
					case <-time.After(5 * time.Millisecond):
						heard2 = false
					}

					if shouldHear != heard2 {
						t.Errorf("Correct operation not recorded "+
							"for listener 2: Expected: %v, Occured: %v",
							shouldHear, heard2)
					}
				}
			}
		}
	}
}

//tests that Unregister removes the listener and only the listener
func TestSwitchboard_Unregister(t *testing.T) {
	sw := New()

	uid := id.NewIdFromUInt(42, id.User, t)
	mt := catalog.MessageType(1)

	l := func(receive Message) {}

	lid1 := sw.RegisterFunc("a", uid, mt, l)

	lid2 := sw.RegisterFunc("a", uid, mt, l)

	sw.Unregister(lid1)

	//get sets to check
	setID := sw.id.Get(uid)
	setType := sw.messageType.Get(mt)

	//check that the removed listener is not registered
	if setID.Has(lid1) {
		t.Errorf("Removed Listener is registered by ID, should not be")
	}

	if setType.Has(lid1) {
		t.Errorf("Removed Listener not registered by Message Tag, " +
			"should not be")
	}

	//check that the not removed listener is still registered
	if !setID.Has(lid2) {
		t.Errorf("Remaining Listener is not registered by ID")
	}

	if !setType.Has(lid2) {
		t.Errorf("Remaining Listener is not registered by Message Tag")
	}
}

// Tests that three listeners with three different message types but the same
// user are deleted with Switchboard.UnregisterUserListeners and that three
// other listeners with the same message types but different users are not
// delete.
func TestSwitchboard_UnregisterUserListeners(t *testing.T) {
	sw := New()

	uid1 := id.NewIdFromUInt(42, id.User, t)
	uid2 := id.NewIdFromUInt(11, id.User, t)

	l := func(receive Message) {}

	lid1 := sw.RegisterFunc("a", uid1, catalog.NoType, l)
	lid2 := sw.RegisterFunc("b", uid1, catalog.XxMessage, l)
	lid3 := sw.RegisterFunc("c", uid1, catalog.KeyExchangeTrigger, l)
	lid4 := sw.RegisterFunc("d", uid2, catalog.NoType, l)
	lid5 := sw.RegisterFunc("e", uid2, catalog.XxMessage, l)
	lid6 := sw.RegisterFunc("f", uid2, catalog.KeyExchangeTrigger, l)

	sw.UnregisterUserListeners(uid1)

	if s, exists := sw.id.list[*uid1]; exists {
		t.Errorf("Set for ID %s still exists: %+v", uid1, s)
	}

	if sw.messageType.Get(lid1.messageType).Has(lid1) {
		t.Errorf("Listener %+v still exists", lid1)
	}
	if sw.messageType.Get(lid2.messageType).Has(lid2) {
		t.Errorf("Listener %+v still exists", lid2)
	}
	if sw.messageType.Get(lid3.messageType).Has(lid3) {
		t.Errorf("Listener %+v still exists", lid3)
	}

	if !sw.messageType.Get(lid4.messageType).Has(lid4) {
		t.Errorf("Listener %+v does not exist", lid4)
	}
	if !sw.messageType.Get(lid5.messageType).Has(lid5) {
		t.Errorf("Listener %+v does not exist", lid5)
	}
	if !sw.messageType.Get(lid6.messageType).Has(lid6) {
		t.Errorf("Listener %+v does not exist", lid6)
	}
}
