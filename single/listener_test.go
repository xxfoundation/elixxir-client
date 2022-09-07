////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	cMixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

type cbReceiver struct {
	requestChan chan *Request
}

func newCbReceiver(requestChan chan *Request) *cbReceiver {
	return &cbReceiver{requestChan: requestChan}
}

func (cr *cbReceiver) Callback(
	r *Request, _ receptionID.EphemeralIdentity, _ []rounds.Round) {
	cr.requestChan <- r
}

// Tests that Listen returns the expected listener.
func TestListen(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	tag := "myTag"
	myID := id.NewIdFromString("myID", id.User, t)
	privKey := grp.NewInt(34)
	handler := newListenMockCmixHandler()

	expected := &listener{
		tag:       tag,
		grp:       grp,
		myID:      myID,
		myPrivKey: privKey,
		net:       newMockListenCmix(handler),
	}

	l := Listen(tag, myID, privKey,
		newMockListenCmix(handler), grp, nil)

	if !reflect.DeepEqual(expected, l) {
		t.Errorf("New Listener does not match expected."+
			"\nexpected: %#v\nreceived: %#v", expected, l)
	}
}

// Tests that listener.process correctly unmarshalls the payload and returns the
// Request with the expected fields on the callback.
func Test_listener_Process(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	privKey := grp.NewInt(42)
	recipient := contact.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: grp.ExpG(privKey, grp.NewInt(1)),
	}
	handler := newListenMockCmixHandler()

	payload := []byte("I am the payload!")
	msg, sendingID, publicKey, dhKey := newRequestMessage(
		payload, grp, recipient, prng, handler, t)

	requestChan := make(chan *Request, 10)
	l := &listener{
		tag:       "tag",
		grp:       grp,
		myID:      id.NewIdFromString("myID", id.User, t),
		myPrivKey: privKey,
		cb:        newCbReceiver(requestChan),
		net:       newMockListenCmix(handler),
	}

	err := l.process(msg, sendingID, rounds.Round{})
	if err != nil {
		t.Errorf("process returned an error: %+v", err)
	}

	used := uint32(0)
	expected := &Request{
		sender:         sendingID.Source,
		senderPubKey:   publicKey,
		dhKey:          dhKey,
		tag:            l.tag,
		maxParts:       6,
		used:           &used,
		requestPayload: payload,
		net:            l.net,
	}

	select {
	case r := <-requestChan:
		if !reflect.DeepEqual(expected, r) {
			t.Errorf("Received unexpected values."+
				"\nexpected: %+v\nreceived: %+v", expected, r)
		}
	case <-time.After(15 * time.Millisecond):
		t.Error("Timed out waiting to receive callback.")
	}
}

// newRequestMessage creates a new encrypted request message for testing.
func newRequestMessage(payload []byte, grp *cyclic.Group,
	recipient contact.Contact, rng io.Reader, handler *mockListenCmixHandler,
	t *testing.T) (format.Message, receptionID.EphemeralIdentity, *cyclic.Int,
	*cyclic.Int) {

	net := newMockListenCmix(handler)
	maxResponseMessages := uint8(6)
	params := GetDefaultRequestParams()
	timeStart := netTime.Now()

	// Generate DH key and public key
	dhKey, publicKey, err := generateDhKeys(grp, recipient.DhPubKey, rng)
	if err != nil {
		t.Errorf("Failed to generate DH keys: %+v", err)
	}

	// Build the message payload
	request := message.NewRequest(
		net.GetMaxMessageLength(), grp.GetP().ByteLen())
	requestPayload := message.NewRequestPayload(
		request.GetPayloadSize(), payload, maxResponseMessages)

	// Generate new user ID and address ID
	var sendingID receptionID.EphemeralIdentity
	requestPayload, sendingID, err = makeIDs(
		requestPayload, publicKey, 8, params.Timeout, timeStart, rng)
	if err != nil {
		t.Errorf("Failed to make new sending ID: %+v", err)
	}

	// Encrypt and assemble payload
	fp := singleUse.NewRequestFingerprint(recipient.DhPubKey)
	key := singleUse.NewRequestKey(dhKey)
	encryptedPayload := auth.Crypt(key, fp[:24], requestPayload.Marshal())

	// Generate cMix message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	// Assemble the payload
	request.SetPubKey(publicKey)
	request.SetPayload(encryptedPayload)

	msg := format.NewMessage(net.numPrimeBytes)
	msg.SetMac(mac)
	msg.SetContents(request.Marshal())
	msg.SetKeyFP(fp)

	return msg, sendingID, publicKey, dhKey
}

// First successfully sends and receives request. Then, once listener.Stop is
// called, the next send is never received.
func Test_listener_Stop(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	grp := cyclic.NewGroup(large.NewInt(173), large.NewInt(2))
	privKey := grp.NewInt(42)
	recipient := contact.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: grp.ExpG(privKey, grp.NewInt(1)),
	}
	handler := newListenMockCmixHandler()
	net := newMockListenCmix(handler)

	payload := []byte("I am the payload!")
	msg, _, _, _ := newRequestMessage(
		payload, grp, recipient, prng, handler, t)

	requestChan := make(chan *Request, 10)
	myID := id.NewIdFromString("myID", id.User, t)
	tag := "tag"
	l := Listen(tag, myID, privKey, net, grp, newCbReceiver(requestChan))

	svc := cMixMsg.Service{
		Identifier: myID[:],
		Tag:        tag,
		Metadata:   myID[:],
	}
	_, _, _ = net.Send(myID, msg.GetKeyFP(), svc, msg.GetContents(),
		msg.GetMac(), cmix.CMIXParams{})

	select {
	case <-requestChan:
	case <-time.After(15 * time.Millisecond):
		t.Error("Timed out waiting to receive callback.")
	}

	l.Stop()
	_, _, _ = net.Send(myID, msg.GetKeyFP(), svc, msg.GetContents(),
		msg.GetMac(), cmix.CMIXParams{})

	select {
	case r := <-requestChan:
		t.Errorf("Received callback when it should have been stopped: %+v", r)
	case <-time.After(15 * time.Millisecond):
	}
}

////////////////////////////////////////////////////////////////////////////////
// Mock cMix                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockListenCmixHandler struct {
	fingerprintMap map[id.ID]map[format.Fingerprint][]cMixMsg.Processor
	serviceMap     map[id.ID]map[string][]cMixMsg.Processor
	sync.Mutex
}

func newListenMockCmixHandler() *mockListenCmixHandler {
	return &mockListenCmixHandler{
		serviceMap: make(map[id.ID]map[string][]cMixMsg.Processor),
	}
}

type mockListenCmix struct {
	numPrimeBytes int
	handler       *mockListenCmixHandler
}

func newMockListenCmix(handler *mockListenCmixHandler) *mockListenCmix {
	return &mockListenCmix{
		numPrimeBytes: 4096,
		handler:       handler,
	}
}

func (m mockListenCmix) GetMaxMessageLength() int {
	return format.NewMessage(m.numPrimeBytes).ContentsSize()
}

func (m mockListenCmix) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service cMixMsg.Service, payload, mac []byte, _ cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetKeyFP(fingerprint)

	m.handler.Lock()
	defer m.handler.Unlock()
	for _, p := range m.handler.serviceMap[*recipient][service.Tag] {
		p.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}
	for _, p := range m.handler.fingerprintMap[*recipient][fingerprint] {
		p.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}

	return 0, ephemeral.Id{}, nil
}

func (m mockListenCmix) GetInstance() *network.Instance {
	// TODO implement me
	panic("implement me")
}

func (m mockListenCmix) AddFingerprint(identity *id.ID, fp format.Fingerprint,
	mp cMixMsg.Processor) error {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.fingerprintMap[*identity]; !exists {
		m.handler.fingerprintMap[*identity] =
			map[format.Fingerprint][]cMixMsg.Processor{fp: {mp}}
		return nil
	} else if _, exists = m.handler.fingerprintMap[*identity][fp]; !exists {
		m.handler.fingerprintMap[*identity][fp] =
			[]cMixMsg.Processor{mp}
		return nil
	}

	m.handler.fingerprintMap[*identity][fp] =
		append(m.handler.fingerprintMap[*identity][fp], mp)
	return nil
}

func (m mockListenCmix) AddService(
	clientID *id.ID, ms cMixMsg.Service, mp cMixMsg.Processor) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.serviceMap[*clientID]; !exists {
		m.handler.serviceMap[*clientID] =
			map[string][]cMixMsg.Processor{ms.Tag: {mp}}
		return
	} else if _, exists = m.handler.serviceMap[*clientID][ms.Tag]; !exists {
		m.handler.serviceMap[*clientID][ms.Tag] =
			[]cMixMsg.Processor{mp}
		return
	}

	m.handler.serviceMap[*clientID][ms.Tag] =
		append(m.handler.serviceMap[*clientID][ms.Tag], mp)
}

func (m mockListenCmix) DeleteService(
	clientID *id.ID, toDelete cMixMsg.Service, processor cMixMsg.Processor) {
	m.handler.Lock()
	defer m.handler.Unlock()

	for i, p := range m.handler.serviceMap[*clientID][toDelete.Tag] {
		if p == processor {
			m.handler.serviceMap[*clientID][toDelete.Tag] =
				remove(m.handler.serviceMap[*clientID][toDelete.Tag], i)
		}
	}
}

func remove(s []cMixMsg.Processor, i int) []cMixMsg.Processor {
	s2 := make([]cMixMsg.Processor, 0)
	s2 = append(s2, s[:i]...)
	return append(s2, s[i+1:]...)
}

func (m mockListenCmix) CheckInProgressMessages() {
	// TODO implement me
	panic("implement me")
}
