////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/single/message"
	"gitlab.com/elixxir/crypto/contact"
	dh "gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

func TestGetMaxRequestSize(t *testing.T) {
}

type mockResponse struct {
	payloadChan chan []byte
}

func (m mockResponse) Callback(
	payload []byte, _ receptionID.EphemeralIdentity, _ []rounds.Round, _ error) {
	m.payloadChan <- payload
}

type mockReceiver struct {
	t           testing.TB
	response    []byte
	requestChan chan *Request
}

func (m *mockReceiver) Callback(
	request *Request, _ receptionID.EphemeralIdentity, _ []rounds.Round) {
	m.requestChan <- request
	_, err := request.Respond(m.response, cmix.GetDefaultCMIXParams(), 0)
	if err != nil {
		m.t.Errorf("Failed to respond: %+v", err)
	}
}

// Tests single-use request and response.
func TestTransmitRequest(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)
	rng := fastRNG.NewStreamGenerator(12, 1024, csprng.NewSystemRNG).GetStream()
	handler := newMockCmixHandler()
	myID := id.NewIdFromString("myID", id.User, t)
	net := newMockCmix(myID, handler, t)
	grp := net.GetInstance().GetE2EGroup()

	partnerPrivKey := dh.GeneratePrivateKey(dh.DefaultPrivateKeyLength, grp, rng)
	partnerPubKey := dh.GeneratePublicKey(partnerPrivKey, grp)

	recipient := contact.Contact{
		ID:       id.NewIdFromString("recipient", id.User, t),
		DhPubKey: partnerPubKey,
	}

	buff := bytes.NewBuffer(nil)
	payloadSize := message.GetRequestPayloadSize(net.GetMaxMessageLength(),
		grp.GetP().ByteLen())
	requestSize := message.GetRequestContentsSize(payloadSize)
	firstPart := make([]byte, requestSize)
	copy(firstPart, "First part payload.")
	buff.Write(firstPart)
	requestPartSize := message.GetRequestPartContentsSize(
		net.GetMaxMessageLength())

	for i := 0; i < 10; i++ {
		part := make([]byte, requestPartSize)
		copy(part, fmt.Sprintf("Part #%d payload.", i))
		buff.Write(part)
	}
	payload := buff.Bytes()

	tag := "TestTransmitRequest"
	responsePayload := make([]byte, 4096)
	copy(responsePayload, "My response.")
	responseChan := make(chan []byte, 10)
	response := mockResponse{responseChan}
	params := GetDefaultRequestParams()

	requestChan := make(chan *Request, 10)
	recipientNet := newMockCmix(recipient.ID, handler, t)
	_ = Listen(tag, recipient.ID, partnerPrivKey, recipientNet, grp,
		&mockReceiver{t, responsePayload, requestChan})

	_, _, err := TransmitRequest(
		recipient, tag, payload, response, params, net, rng, grp)
	if err != nil {
		t.Errorf("TransmitRequest returned an error: %+v", err)
	}

	select {
	case r := <-requestChan:
		if !bytes.Equal(r.GetPayload(), payload) {
			t.Errorf("Received unexpected request payload."+
				"\nexpected: %q\nreceived: %q", payload, r.GetPayload())
		}
	case <-time.After(30 * time.Millisecond):
		t.Errorf("Timed out waiting to receive request.")
	}

	select {
	case r := <-responseChan:
		if !bytes.Equal(r, responsePayload) {
			t.Errorf("Received unexpected response.\nexpected: %q\nreceived: %q",
				payload, r)
		}
	case <-time.After(30 * time.Millisecond):
		t.Errorf("Timed out waiting to receive response.")
	}
}

// Tests that waitForTimeout returns and does not call the callback when the
// kill channel is used.
func Test_waitForTimeout(t *testing.T) {
	timeout := 300 * time.Millisecond
	cbChan := make(chan error, 1)
	cb := func(
		_ []byte, _ receptionID.EphemeralIdentity, _ []rounds.Round, err error) {
		cbChan <- err
	}
	killChan := make(chan bool, 1)

	go func() {
		time.Sleep(timeout / 2)
		killChan <- true
	}()

	waitForTimeout(killChan, cb, timeout)

	select {
	case <-cbChan:
		t.Error("Callback called when waitForTimeout should have been killed.")
	case <-time.After(timeout):
	}
}

// Error path: tests that waitForTimeout returns an error on the callback when
// the timeout is reached.
func Test_waitForTimeout_TimeoutError(t *testing.T) {
	timeout := 15 * time.Millisecond
	expectedErr := fmt.Sprintf(errResponseTimeout, timeout)
	cbChan := make(chan error)
	cb := func(
		_ []byte, _ receptionID.EphemeralIdentity, _ []rounds.Round, err error) {
		cbChan <- err
	}
	killChan := make(chan bool)

	go waitForTimeout(killChan, cb, timeout)

	select {
	case r := <-cbChan:
		if r == nil || r.Error() != expectedErr {
			t.Errorf("Did not get expected error on callback."+
				"\nexpected: %s\nreceived: %+v", expectedErr, r)
		}
	case <-time.After(timeout * 2):
		t.Errorf("Timed out waiting on callback.")
	}
}

// Builds a payload alongside the expected first part and list of subsequent
// parts and tests that partitionPayload properly partitions the payload into
// the expected parts.
func Test_partitionPayload(t *testing.T) {
	const partSize = 16
	expectedFirstPart := []byte("first part")
	expectedParts := make([][]byte, 10)
	payload := bytes.NewBuffer(expectedFirstPart)
	for i := range expectedParts {
		expectedParts[i] = make([]byte, partSize)
		copy(expectedParts[i], fmt.Sprintf("Part #%d", i))
		payload.Write(expectedParts[i])
	}

	firstPart, parts := partitionPayload(
		len(expectedFirstPart), partSize, payload.Bytes())

	if !bytes.Equal(expectedFirstPart, firstPart) {
		t.Errorf("Received unexpected first part.\nexpected: %q\nreceived: %q",
			expectedFirstPart, firstPart)
	}

	if !reflect.DeepEqual(expectedParts, parts) {
		t.Errorf("Received unexpected parts.\nexpected: %q\nreceived: %q",
			expectedParts, parts)
	}
}
