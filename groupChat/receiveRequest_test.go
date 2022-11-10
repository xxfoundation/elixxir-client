////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/v5/catalog"
	sessionImport "gitlab.com/elixxir/client/v5/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v5/e2e/receive"
	gs "gitlab.com/elixxir/client/v5/groupChat/groupStore"
	util "gitlab.com/elixxir/client/v5/storage/utility"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests that the correct group is received from the request.
func TestRequestListener_Hear(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, _ := newTestManagerWithStore(prng, 10, 0, requestFunc, t)
	g := newTestGroupWithUser(m.getE2eGroup(),
		m.getReceptionIdentity().ID, m.getE2eHandler().GetHistoricalDHPubkey(),
		m.getE2eHandler().GetHistoricalDHPrivkey(), prng, t)

	requestMarshaled, err := proto.Marshal(&Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
		Created:     g.Created.UnixNano(),
	})
	if err != nil {
		t.Errorf("Failed to marshal proto message: %+v", err)
	}

	msg := receive.Message{
		Sender:      g.Members[0].ID,
		Payload:     requestMarshaled,
		MessageType: catalog.GroupCreationRequest,
	}
	listener := requestListener{m: m}

	myVariant := sidh.KeyVariantSidhA
	mySIDHPrivKey := util.NewSIDHPrivateKey(myVariant)
	mySIDHPubKey := util.NewSIDHPublicKey(myVariant)
	_ = mySIDHPrivKey.Generate(prng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	theirVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	theirSIDHPrivKey := util.NewSIDHPrivateKey(theirVariant)
	theirSIDHPubKey := util.NewSIDHPublicKey(theirVariant)
	_ = theirSIDHPrivKey.Generate(prng)
	theirSIDHPrivKey.GeneratePublicKey(theirSIDHPubKey)

	_, _ = m.getE2eHandler().AddPartner(
		g.Members[0].ID,
		g.Members[0].DhKey,
		m.getE2eHandler().GetHistoricalDHPrivkey(),
		theirSIDHPubKey, mySIDHPrivKey,
		sessionImport.GetDefaultParams(),
		sessionImport.GetDefaultParams(),
	)

	go listener.Hear(msg)

	select {
	case receivedGrp := <-requestChan:
		if !reflect.DeepEqual(g, receivedGrp) {
			t.Errorf("receiveRequest() failed to return the expected group."+
				"\nexpected: %#v\nreceived: %#v", g, receivedGrp)
		}
	case <-time.NewTimer(5 * time.Millisecond).C:
		t.Error("Timed out while waiting for callback.")
	}
}

// Tests that the callback is not called when the group already exists in the
// manager.
func TestRequestListener_Hear_GroupExists(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, g := newTestManagerWithStore(prng, 10, 0, requestFunc, t)

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

	listener := requestListener{m: m}

	msg := receive.Message{
		Payload:     requestMarshaled,
		MessageType: catalog.GroupCreationRequest,
	}

	go listener.Hear(msg)

	select {
	case <-requestChan:
		t.Error("receiveRequest() called the callback when the group already " +
			"exists in the list.")
	case <-time.NewTimer(5 * time.Millisecond).C:
	}
}

// Tests that the callback is not called when the send message is not of the
// correct type.
func TestRequestListener_Hear_BadMessageType(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	requestChan := make(chan gs.Group)
	requestFunc := func(g gs.Group) { requestChan <- g }
	m, _ := newTestManagerWithStore(prng, 10, 0, requestFunc, t)

	msg := receive.Message{
		MessageType: catalog.NoType,
	}

	listener := requestListener{m: m}

	go listener.Hear(msg)

	select {
	case receivedGrp := <-requestChan:
		t.Errorf("Callback called when the message should have been skipped: %#v",
			receivedGrp)
	case <-time.NewTimer(5 * time.Millisecond).C:
	}
}

// Unit test of readRequest.
func Test_manager_readRequest(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManager(t)

	myVariant := sidh.KeyVariantSidhA
	mySIDHPrivKey := util.NewSIDHPrivateKey(myVariant)
	mySIDHPubKey := util.NewSIDHPublicKey(myVariant)
	_ = mySIDHPrivKey.Generate(prng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	theirVariant := sidh.KeyVariant(sidh.KeyVariantSidhB)
	theirSIDHPrivKey := util.NewSIDHPrivateKey(theirVariant)
	theirSIDHPubKey := util.NewSIDHPublicKey(theirVariant)
	_ = theirSIDHPrivKey.Generate(prng)
	theirSIDHPrivKey.GeneratePublicKey(theirSIDHPubKey)

	_, _ = m.getE2eHandler().AddPartner(
		g.Members[0].ID,
		g.Members[0].DhKey,
		m.getE2eHandler().GetHistoricalDHPrivkey(),
		theirSIDHPubKey, mySIDHPrivKey,
		sessionImport.GetDefaultParams(),
		sessionImport.GetDefaultParams(),
	)

	requestMarshaled, err := proto.Marshal(&Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
		Created:     g.Created.UnixNano(),
	})
	if err != nil {
		t.Errorf("Failed to marshal proto message: %+v", err)
	}

	msg := receive.Message{
		Payload:     requestMarshaled,
		MessageType: catalog.GroupCreationRequest,
	}

	newGrp, err := m.readRequest(msg)
	if err != nil {
		t.Errorf("readRequest() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(g, newGrp) {
		t.Errorf("readRequest() returned the wrong group."+
			"\nexpected: %#v\nreceived: %#v", g, newGrp)
	}
}

// Error path: an error is returned if the message type is incorrect.
func Test_manager_readRequest_MessageTypeError(t *testing.T) {
	m, _ := newTestManager(t)
	expectedErr := sendMessageTypeErr
	msg := receive.Message{
		MessageType: catalog.NoType,
	}

	_, err := m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned if the proto message cannot be unmarshalled.
func Test_manager_readRequest_ProtoUnmarshalError(t *testing.T) {
	expectedErr := strings.SplitN(deserializeMembershipErr, "%", 2)[0]
	m, _ := newTestManager(t)

	requestMarshaled, err := proto.Marshal(&Request{
		Members: []byte("Invalid membership serial."),
	})
	if err != nil {
		t.Errorf("Failed to marshal proto message: %+v", err)
	}

	msg := receive.Message{
		Payload:     requestMarshaled,
		MessageType: catalog.GroupCreationRequest,
	}

	_, err = m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: an error is returned if the membership cannot be deserialized.
func Test_manager_readRequest_DeserializeMembershipError(t *testing.T) {
	m, _ := newTestManager(t)
	expectedErr := strings.SplitN(protoUnmarshalErr, "%", 2)[0]
	msg := receive.Message{
		Payload:     []byte("Invalid message."),
		MessageType: catalog.GroupCreationRequest,
	}

	_, err := m.readRequest(msg)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("readRequest() did not return the expected error."+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}
