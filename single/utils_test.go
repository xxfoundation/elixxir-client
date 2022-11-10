////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/v5/cmix"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/message"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Mock cMix                                                           //
////////////////////////////////////////////////////////////////////////////////

// Tests that mockCmix adheres to the Cmix interface.
var _ Cmix = (*mockCmix)(nil)

type mockCmixHandler struct {
	fingerprints map[id.ID]map[format.Fingerprint]message.Processor
	services     map[id.ID]map[string]message.Processor
	sends        []send
	sync.Mutex
}

func newMockCmixHandler() *mockCmixHandler {
	return &mockCmixHandler{
		fingerprints: make(map[id.ID]map[format.Fingerprint]message.Processor),
		services:     make(map[id.ID]map[string]message.Processor),
		sends:        []send{},
	}
}

type mockCmix struct {
	myID             *id.ID
	numPrimeBytes    int
	addressSpaceSize uint8
	health           bool
	instance         *network.Instance
	handler          *mockCmixHandler
	t                testing.TB
	sync.Mutex
}

type send struct {
	myID      *id.ID
	recipient *id.ID
	ms        message.Service
	msg       format.Message
}

func newMockCmix(myID *id.ID, handler *mockCmixHandler, t testing.TB) *mockCmix {
	comms := &connect.ProtoComms{Manager: connect.NewManagerTesting(t)}
	def := getNDF()

	instance, err := network.NewInstanceTesting(comms, def, def, nil, nil, t)
	if err != nil {
		panic(err)
	}

	return &mockCmix{
		myID:             myID,
		numPrimeBytes:    1024,
		addressSpaceSize: 18,
		health:           true,
		instance:         instance,
		handler:          handler,
		t:                t,
	}
}

func (m *mockCmix) IsHealthy() bool {
	return m.health
}

func (m *mockCmix) GetMaxMessageLength() int {
	msg := format.NewMessage(m.numPrimeBytes)
	return msg.ContentsSize()
}

func (m *mockCmix) GetAddressSpace() uint8 {
	return m.addressSpaceSize
}

func (m *mockCmix) DeleteClientFingerprints(identity *id.ID) {
	m.handler.Lock()
	defer m.handler.Unlock()
	delete(m.handler.fingerprints, *identity)
}

func (m *mockCmix) AddFingerprint(
	identity *id.ID, fp format.Fingerprint, mp message.Processor) error {
	m.handler.Lock()
	defer m.handler.Unlock()
	if _, exists := m.handler.fingerprints[*identity]; !exists {
		m.handler.fingerprints[*identity] =
			map[format.Fingerprint]message.Processor{fp: mp}
	} else {
		m.handler.fingerprints[*identity][fp] = mp
	}
	return nil
}

func (m *mockCmix) AddIdentity(*id.ID, time.Time, bool) {}

func (m *mockCmix) Send(recipient *id.ID, fp format.Fingerprint,
	ms message.Service, payload, mac []byte, _ cmix.CMIXParams) (
	rounds.Round, ephemeral.Id, error) {

	msg := format.NewMessage(m.numPrimeBytes)
	msg.SetMac(mac)
	msg.SetKeyFP(fp)
	msg.SetContents(payload)

	m.handler.Lock()
	defer m.handler.Unlock()

	var sent bool

	if _, exists := m.handler.fingerprints[*recipient]; exists {
		if mp, exists := m.handler.fingerprints[*recipient][fp]; exists {
			sent = true
			go mp.Process(
				msg, receptionID.EphemeralIdentity{Source: m.myID}, rounds.Round{})
		}
	}

	if _, exists := m.handler.services[*recipient]; exists {
		if mp, exists := m.handler.services[*recipient][serviceKey(ms)]; exists {
			sent = true
			go mp.Process(
				msg, receptionID.EphemeralIdentity{Source: m.myID}, rounds.Round{})
		}
	}

	if !sent {
		m.handler.sends = append(m.handler.sends, send{
			myID:      m.myID,
			recipient: recipient,
			ms:        ms,
			msg:       msg,
		})
	}

	return rounds.Round{}, ephemeral.Id{}, nil
}

func serviceKey(ms message.Service) string {
	return string(append(ms.Identifier, []byte(ms.Tag)...))
}

func (m *mockCmix) AddService(clientID *id.ID, ms message.Service, mp message.Processor) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.services[*clientID]; !exists {
		m.handler.services[*clientID] = map[string]message.Processor{
			serviceKey(ms): mp}
	} else {
		m.handler.services[*clientID][serviceKey(ms)] = mp
	}
}

func (m *mockCmix) DeleteService(clientID *id.ID, ms message.Service, _ message.Processor) {
	m.handler.Lock()
	defer m.handler.Unlock()

	if _, exists := m.handler.services[*clientID]; exists {
		delete(m.handler.services[*clientID], serviceKey(ms))
	}
}

func (m *mockCmix) GetInstance() *network.Instance {
	return m.instance
}

func (m *mockCmix) CheckInProgressMessages() {
	m.handler.Lock()
	defer m.handler.Unlock()

	var newSends []send

	for _, s := range m.handler.sends {
		var sent bool

		if _, exists := m.handler.fingerprints[*s.recipient]; exists {
			if mp, exists := m.handler.fingerprints[*s.recipient][s.msg.GetKeyFP()]; exists {
				sent = true
				go mp.Process(
					s.msg, receptionID.EphemeralIdentity{Source: s.myID}, rounds.Round{})
			}
		}

		if _, exists := m.handler.services[*s.recipient]; exists {
			if mp, exists := m.handler.services[*s.recipient][serviceKey(s.ms)]; exists {
				sent = true
				go mp.Process(
					s.msg, receptionID.EphemeralIdentity{Source: s.myID}, rounds.Round{})
			}
		}

		if !sent {
			newSends = append(newSends, s)
		}
	}

	m.handler.sends = newSends
}
