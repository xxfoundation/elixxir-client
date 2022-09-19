////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"

	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
)

// newRsaPubKey generates a new random RSA public key for testing.
func newRsaPubKey(seed int64, t *testing.T) *rsa.PublicKey {
	prng := rand.New(rand.NewSource(seed))
	privKey, err := rsa.GenerateKey(prng, 64)
	if err != nil {
		t.Errorf("Failed to generate new RSA key: %+v", err)
	}

	return privKey.GetPublic()
}

////////////////////////////////////////////////////////////////////////////////
// Mock cMix                                                           //
////////////////////////////////////////////////////////////////////////////////

type mockCmixHandler struct {
	processorMap map[id.ID]map[string][]message.Processor
	sync.Mutex
}

func newMockCmixHandler() *mockCmixHandler {
	return &mockCmixHandler{
		processorMap: make(map[id.ID]map[string][]message.Processor),
	}
}

type mockCmix struct {
	numPrimeBytes int
	health        bool
	handler       *mockCmixHandler
}

func newMockCmix(handler *mockCmixHandler) *mockCmix {
	return &mockCmix{
		numPrimeBytes: 4096,
		health:        true,
		handler:       handler,
	}
}

func (m *mockCmix) GetMaxMessageLength() int {
	return format.NewMessage(m.numPrimeBytes * 2).ContentsSize()
}

func (m *mockCmix) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {

	fingerprint, service, payload, mac, err := assembler(42)
	if err != nil {
		panic(err)
	}

	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetKeyFP(fingerprint)

	m.handler.Lock()
	defer m.handler.Unlock()

	for _, p := range m.handler.processorMap[*recipient][service.Tag] {
		p.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}

	return rounds.Round{}, ephemeral.Id{}, nil
}

func (m *mockCmix) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service, payload, mac []byte, _ cmix.CMIXParams) (
	id.Round, ephemeral.Id, error) {
	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetKeyFP(fingerprint)

	m.handler.Lock()
	defer m.handler.Unlock()
	for _, p := range m.handler.processorMap[*recipient][service.Tag] {
		p.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}

	return 0, ephemeral.Id{}, nil
}

func (m *mockCmix) IsHealthy() bool {
	return m.health
}

func (m *mockCmix) AddIdentity(id *id.ID, _ time.Time, _ bool) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.processorMap[*id]; exists {
		return
	}

	m.handler.processorMap[*id] = make(map[string][]message.Processor)
}

func (m *mockCmix) AddService(clientID *id.ID, newService message.Service,
	response message.Processor) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.processorMap[*clientID][newService.Tag]; !exists {
		m.handler.processorMap[*clientID][newService.Tag] =
			[]message.Processor{response}
		return
	}

	m.handler.processorMap[*clientID][newService.Tag] =
		append(m.handler.processorMap[*clientID][newService.Tag], response)
}

func (m *mockCmix) DeleteClientService(clientID *id.ID) {
	m.handler.Lock()
	defer m.handler.Unlock()

	for tag := range m.handler.processorMap[*clientID] {
		delete(m.handler.processorMap[*clientID], tag)
	}
}

func (m *mockCmix) RemoveIdentity(id *id.ID) {
	m.handler.Lock()
	defer m.handler.Unlock()

	delete(m.handler.processorMap, *id)
}
