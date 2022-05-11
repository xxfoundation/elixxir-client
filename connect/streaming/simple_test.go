package streaming

import (
	"bytes"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	ce2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

var pub = "-----BEGIN CERTIFICATE-----\n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\n-----END CERTIFICATE-----\n"

// Mock connect interface
type mockConnect struct {
	sendChan     chan []byte
	responseChan chan []byte
	unregister   chan bool
}

// Closer deletes this Connection's partner.Manager and releases resources
func (mc *mockConnect) Close() error {
	return nil
}

// GetPartner returns the partner.Manager for this Connection
func (mc *mockConnect) GetPartner() partner.Manager {
	return nil
}

// SendE2E is a wrapper for sending specifically to the Connection's partner.Manager
func (mc *mockConnect) SendE2E(mt catalog.MessageType, payload []byte, params e2e.Params) (
	[]id.Round, ce2e.MessageID, time.Time, error) {
	mc.sendChan <- payload
	return []id.Round{2}, ce2e.MessageID{1}, time.Now(), nil
}

// RegisterListener is used for E2E reception
// and allows for reading data sent from the partner.Manager
func (mc *mockConnect) RegisterListener(messageType catalog.MessageType,
	newListener receive.Listener) receive.ListenerID {
	go func() {
		select {
		case p := <-mc.responseChan:
			newListener.Hear(receive.Message{Payload: p})
		case <-mc.unregister:
			return
		}
	}()
	return receive.ListenerID{}
}

// Unregister listener for E2E reception
func (mc *mockConnect) Unregister(listenerID receive.ListenerID) {
	mc.unregister <- true
}

// Test for simple streaming write implementation
func TestSimple_Write(t *testing.T) {
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte, 1),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := []byte("hello from me")
	bytesWritten, err := ss.Write(data)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	}
	if bytesWritten != len(data) {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", len(data), bytesWritten)
	}
	timeout := time.Tick(time.Second)
	select {
	case p := <-mc.sendChan:
		t.Logf("Received payload %+v over sendChan", p)
		if bytes.Compare(p, data) != 0 {
			t.Errorf("Did not receive expected bytes\n\tExpected: %+v\n\tReceived: %+v\n", data, p)
		}
	case <-timeout:
		t.Errorf("Timed out waiting for send")
	}
}

// Unit test for Read, passing in array of same size as data in buffer
func TestSimple_Read_sameSize(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	l := 32
	data := make([]byte, l)
	n0, err := rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	time.Sleep(time.Second)
	receivedData := make([]byte, l)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != n0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", n0, n)
	} else if bytes.Compare(receivedData, data) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data, receivedData)
	}

	receivedDataRest := make([]byte, l)
	n, err = ss.Read(receivedDataRest)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != 0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedDataRest, make([]byte, l)) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[l:], receivedDataRest)
	}
}

// Unit test for Read, passing in array larger than size of data in buffer
func TestSimple_Read_larger(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	l := 32
	data := make([]byte, l)
	n0, err := rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	receivedData := make([]byte, l*2)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != n0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", n0, n)
	} else if bytes.Compare(receivedData[:l], data) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data, receivedData)
	}

	receivedDataRest := make([]byte, l*2)
	n, err = ss.Read(receivedDataRest)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != 0 {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedDataRest, make([]byte, l*2)) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[l:], receivedDataRest)
	}
}

// Unit test for Read, passing in array smaller than size of data in buffer
func TestSimple_Read_smaller(t *testing.T) {
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)
	mc := &mockConnect{
		sendChan:     make(chan []byte, 1),
		responseChan: make(chan []byte),
		unregister:   make(chan bool, 1),
	}
	ss, err := NewStream(mc, Params{
		E2E: e2e.GetDefaultParams(),
	})
	if err != nil {
		t.Fatalf("Failed to create simple stream: %+v", err)
	}

	data := make([]byte, 32)
	_, err = rng.GetStream().Read(data)
	if err != nil {
		t.Errorf("Failed to read random data to bytes: %+v", err)
	}

	timeout := time.NewTicker(time.Second)
	select {
	case mc.responseChan <- data:
		t.Logf("Sent data %+v over responseChan", data)
	case <-timeout.C:
		t.Errorf("Timed out sending over response chan")
	}

	l := len(data) / 2
	receivedData := make([]byte, l)
	n, err := ss.Read(receivedData)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != l {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedData, data[:l]) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[:l], receivedData)
	}

	receivedDataRest := make([]byte, l)
	n, err = ss.Read(receivedDataRest)
	if err != nil {
		t.Errorf("Failed to write to simple stream: %+v", err)
	} else if n != l {
		t.Errorf("Did not receive expected bytes written\n\tExpected: %d\n\tReceived: %d\n", l, n)
	} else if bytes.Compare(receivedDataRest, data[l:]) != 0 {
		t.Errorf("Did not receive expected bytes over receiveChan\n\tExpected: %+v\n\tReceived: %+v\n", data[l:], receivedDataRest)
	}
}
