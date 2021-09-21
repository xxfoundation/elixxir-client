///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/golang/protobuf/proto"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// // Tests that the correct group is received from the request.
// func TestManager_receiveRequest(t *testing.T) {
// 	prng := rand.New(rand.NewSource(42))
// 	requestChan := make(chan gs.Group)
// 	requestFunc := func(g gs.Group) { requestChan <- g }
// 	m, _ := newTestManagerWithStore(prng, 10, 0, requestFunc, nil, t)
// 	g := newTestGroupWithUser(m.store.E2e().GetGroup(),
// 		m.store.GetUser().ReceptionID, m.store.GetUser().E2eDhPublicKey,
// 		m.store.GetUser().E2eDhPrivateKey, prng, t)
//
// 	requestMarshaled, err := proto.Marshal(&Request{
// 		Name:        g.Name,
// 		IdPreimage:  g.IdPreimage.Bytes(),
// 		KeyPreimage: g.KeyPreimage.Bytes(),
// 		Members:     g.Members.Serialize(),
// 		Message:     g.InitMessage,
// 	})
// 	if err != nil {
// 		t.Errorf("Failed to marshal proto message: %+v", err)
// 	}
//
// 	msg := message.Receive{
// 		Payload:     requestMarshaled,
// 		MessageType: message.GroupCreationRequest,
// 	}
//
// 	rawMessages := make(chan message.Receive)
// 	quit := make(chan struct{})
// 	go m.receiveRequest(rawMessages, quit)
// 	rawMessages <- msg
//
// 	select {
// 	case receivedGrp := <-requestChan:
// 		if !reflect.DeepEqual(g, receivedGrp) {
// 			t.Errorf("receiveRequest() failed to return the expected group."+
// 				"\nexpected: %#v\nreceived: %#v", g, receivedGrp)
// 		}
// 	case <-time.NewTimer(5 * time.Millisecond).C:
// 		t.Error("Timed out while waiting for callback.")
// 	}
// }

// Tests that the callback is not called when the group already exists in the
// manager.
func TestManager_receiveRequest_GroupExists(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, g := newTestManagerWithStore(prng, 10, 0, requestFunc, nil, t)

	requestMarshaled, err := proto.Marshal(&Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
	})
	if err != nil {
		t.Errorf("Failed to marshal proto message: %+v", err)
	}

	msg := message.Receive{
		Payload:     requestMarshaled,
		MessageType: message.GroupCreationRequest,
	}

	rawMessages := make(chan message.Receive)
	stop := stoppable.NewSingle("testStoppable")
	go m.receiveRequest(rawMessages, stop)
	rawMessages <- msg

	select {
	case <-requestChan:
		t.Error("receiveRequest() called the callback when the group already " +
			"exists in the list.")
	case <-time.NewTimer(5 * time.Millisecond).C:
	}
}

// Tests that the quit channel quits the worker.
func TestManager_receiveRequest_QuitChan(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, _ := newTestManagerWithStore(prng, 10, 0, requestFunc, nil, t)

	rawMessages := make(chan message.Receive)
	stop := stoppable.NewSingle("testStoppable")
	done := make(chan struct{})
	go func() {
		m.receiveRequest(rawMessages, stop)
		done <- struct{}{}
	}()
	if err := stop.Close(); err != nil {
		t.Errorf("Failed to signal close to process: %+v", err)
	}

	select {
	case <-done:
	case <-time.NewTimer(5 * time.Millisecond).C:
		t.Error("receiveRequest() failed to close when the quit.")
	}
}

// Tests that the callback is not called when the send message is not of the
// correct type.
func TestManager_receiveRequest_SendMessageTypeError(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, _ := newTestManagerWithStore(prng, 10, 0, requestFunc, nil, t)

	msg := message.Receive{
		MessageType: message.NoType,
	}

	rawMessages := make(chan message.Receive)
	stop := stoppable.NewSingle("singleStoppable")
	go m.receiveRequest(rawMessages, stop)
	rawMessages <- msg

	select {
	case receivedGrp := <-requestChan:
		t.Errorf("Callback called when the message should have been skipped: %#v",
			receivedGrp)
	case <-time.NewTimer(5 * time.Millisecond).C:
	}
}

// // Unit test of readRequest.
// func TestManager_readRequest(t *testing.T) {
// 	m, g := newTestManager(rand.New(rand.NewSource(42)), t)
// 	_ = m.store.E2e().AddPartner(
// 		g.Members[0].ID,
// 		g.Members[0].DhKey,
// 		m.store.E2e().GetGroup().NewInt(43),
// 		params.GetDefaultE2ESessionParams(),
// 		params.GetDefaultE2ESessionParams(),
// 	)
//
// 	requestMarshaled, err := proto.Marshal(&Request{
// 		Name:        g.Name,
// 		IdPreimage:  g.IdPreimage.Bytes(),
// 		KeyPreimage: g.KeyPreimage.Bytes(),
// 		Members:     g.Members.Serialize(),
// 		Message:     g.InitMessage,
// 	})
// 	if err != nil {
// 		t.Errorf("Failed to marshal proto message: %+v", err)
// 	}
//
// 	msg := message.Receive{
// 		Payload:     requestMarshaled,
// 		MessageType: message.GroupCreationRequest,
// 	}
//
// 	newGrp, err := m.readRequest(msg)
// 	if err != nil {
// 		t.Errorf("readRequest() returned an error: %+v", err)
// 	}
//
// 	if !reflect.DeepEqual(g, newGrp) {
// 		t.Errorf("readRequest() returned the wrong group."+
// 			"\nexpected: %#v\nreceived: %#v", g, newGrp)
// 	}
// }

// Error path: an error is returned if the message type is incorrect.
func TestManager_readRequest_MessageTypeError(t *testing.T) {
	m, _ := newTestManager(rand.New(rand.NewSource(42)), t)
	expectedErr := sendMessageTypeErr
	msg := message.Receive{
		MessageType: message.NoType,
	}

	_, err := m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned if the proto message cannot be unmarshalled.
func TestManager_readRequest_ProtoUnmarshalError(t *testing.T) {
	expectedErr := strings.SplitN(deserializeMembershipErr, "%", 2)[0]
	m, _ := newTestManager(rand.New(rand.NewSource(42)), t)

	requestMarshaled, err := proto.Marshal(&Request{
		Members: []byte("Invalid membership serial."),
	})
	if err != nil {
		t.Errorf("Failed to marshal proto message: %+v", err)
	}

	msg := message.Receive{
		Payload:     requestMarshaled,
		MessageType: message.GroupCreationRequest,
	}

	_, err = m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned if the membership cannot be deserialized.
func TestManager_readRequest_DeserializeMembershipError(t *testing.T) {
	m, _ := newTestManager(rand.New(rand.NewSource(42)), t)
	expectedErr := strings.SplitN(protoUnmarshalErr, "%", 2)[0]
	msg := message.Receive{
		Payload:     []byte("Invalid message."),
		MessageType: message.GroupCreationRequest,
	}

	_, err := m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}