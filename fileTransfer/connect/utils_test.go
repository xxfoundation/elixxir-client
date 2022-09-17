////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	cryptoE2e "gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"sync"
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Mock xxdk.E2e                                                              //
////////////////////////////////////////////////////////////////////////////////

type mockUser struct {
	rid xxdk.ReceptionIdentity
	c   cmix.Client
	s   storage.Session
	rng *fastRNG.StreamGenerator
}

func newMockUser(rid *id.ID, c cmix.Client, s storage.Session,
	rng *fastRNG.StreamGenerator) *mockUser {
	return &mockUser{
		rid: xxdk.ReceptionIdentity{ID: rid},
		c:   c,
		s:   s,
		rng: rng,
	}
}

func (m *mockUser) GetStorage() storage.Session                  { return m.s }
func (m *mockUser) GetReceptionIdentity() xxdk.ReceptionIdentity { return m.rid }
func (m *mockUser) GetCmix() cmix.Client                         { return m.c }
func (m *mockUser) GetRng() *fastRNG.StreamGenerator             { return m.rng }
func (m *mockUser) GetE2E() e2e.Handler                          { return nil }

////////////////////////////////////////////////////////////////////////////////
// Mock cMix                                                                  //
////////////////////////////////////////////////////////////////////////////////

type mockCmixHandler struct {
	sync.Mutex
	processorMap map[format.Fingerprint]message.Processor
}

func newMockCmixHandler() *mockCmixHandler {
	return &mockCmixHandler{
		processorMap: make(map[format.Fingerprint]message.Processor),
	}
}

type mockCmix struct {
	myID          *id.ID
	numPrimeBytes int
	health        bool
	handler       *mockCmixHandler
	healthCBs     map[uint64]func(b bool)
	healthIndex   uint64
	sync.Mutex
}

func newMockCmix(
	myID *id.ID, handler *mockCmixHandler, storage *mockStorage) *mockCmix {
	return &mockCmix{
		myID:          myID,
		numPrimeBytes: storage.GetCmixGroup().GetP().ByteLen(),
		health:        true,
		handler:       handler,
		healthCBs:     make(map[uint64]func(b bool)),
		healthIndex:   0,
	}
}

func (m *mockCmix) Follow(cmix.ClientErrorReport) (stoppable.Stoppable, error) { panic("implement me") }

func (m *mockCmix) GetMaxMessageLength() int {
	msg := format.NewMessage(m.numPrimeBytes)
	return msg.ContentsSize()
}

func (m *mockCmix) Send(*id.ID, format.Fingerprint, message.Service, []byte,
	[]byte, cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	panic("implement me")
}

func (m *mockCmix) SendMany(messages []cmix.TargetedCmixMessage,
	_ cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	m.handler.Lock()
	for _, targetedMsg := range messages {
		msg := format.NewMessage(m.numPrimeBytes)
		msg.SetContents(targetedMsg.Payload)
		msg.SetMac(targetedMsg.Mac)
		msg.SetKeyFP(targetedMsg.Fingerprint)
		m.handler.processorMap[targetedMsg.Fingerprint].Process(msg,
			receptionID.EphemeralIdentity{Source: targetedMsg.Recipient},
			rounds.Round{ID: 42})
	}
	m.handler.Unlock()
	return 42, []ephemeral.Id{}, nil
}

func (m *mockCmix) AddIdentity(*id.ID, time.Time, bool)            { panic("implement me") }
func (m *mockCmix) RemoveIdentity(*id.ID)                          { panic("implement me") }
func (m *mockCmix) GetIdentity(*id.ID) (identity.TrackedID, error) { panic("implement me") }

func (m *mockCmix) AddFingerprint(_ *id.ID, fp format.Fingerprint, mp message.Processor) error {
	m.Lock()
	defer m.Unlock()
	m.handler.processorMap[fp] = mp
	return nil
}

func (m *mockCmix) DeleteFingerprint(_ *id.ID, fp format.Fingerprint) {
	m.handler.Lock()
	delete(m.handler.processorMap, fp)
	m.handler.Unlock()
}

func (m *mockCmix) DeleteClientFingerprints(*id.ID)                          { panic("implement me") }
func (m *mockCmix) AddService(*id.ID, message.Service, message.Processor)    { panic("implement me") }
func (m *mockCmix) DeleteService(*id.ID, message.Service, message.Processor) { panic("implement me") }
func (m *mockCmix) DeleteClientService(*id.ID)                               { panic("implement me") }
func (m *mockCmix) TrackServices(message.ServicesTracker)                    { panic("implement me") }
func (m *mockCmix) CheckInProgressMessages()                                 {}
func (m *mockCmix) IsHealthy() bool                                          { return m.health }
func (m *mockCmix) WasHealthy() bool                                         { return true }

func (m *mockCmix) AddHealthCallback(f func(bool)) uint64 {
	m.Lock()
	defer m.Unlock()
	m.healthIndex++
	m.healthCBs[m.healthIndex] = f
	go f(true)
	return m.healthIndex
}

func (m *mockCmix) RemoveHealthCallback(healthID uint64) {
	m.Lock()
	defer m.Unlock()
	if _, exists := m.healthCBs[healthID]; !exists {
		jww.FATAL.Panicf("No health callback with ID %d exists.", healthID)
	}
	delete(m.healthCBs, healthID)
}

func (m *mockCmix) HasNode(*id.ID) bool            { panic("implement me") }
func (m *mockCmix) NumRegisteredNodes() int        { panic("implement me") }
func (m *mockCmix) TriggerNodeRegistration(*id.ID) { panic("implement me") }

func (m *mockCmix) GetRoundResults(_ time.Duration,
	roundCallback cmix.RoundEventCallback, _ ...id.Round) error {
	go roundCallback(true, false, map[id.Round]cmix.RoundResult{42: {}})
	return nil
}

func (m *mockCmix) LookupHistoricalRound(id.Round, rounds.RoundResultCallback) error {
	panic("implement me")
}
func (m *mockCmix) SendToAny(func(host *connect.Host) (interface{}, error), *stoppable.Single) (interface{}, error) {
	panic("implement me")
}
func (m *mockCmix) SendToPreferred([]*id.ID, gateway.SendToPreferredFunc, *stoppable.Single, time.Duration) (interface{}, error) {
	panic("implement me")
}
func (m *mockCmix) SetGatewayFilter(gateway.Filter)   { panic("implement me") }
func (m *mockCmix) GetHostParams() connect.HostParams { panic("implement me") }
func (m *mockCmix) GetAddressSpace() uint8            { panic("implement me") }
func (m *mockCmix) RegisterAddressSpaceNotification(string) (chan uint8, error) {
	panic("implement me")
}
func (m *mockCmix) UnregisterAddressSpaceNotification(string) { panic("implement me") }
func (m *mockCmix) GetInstance() *network.Instance            { panic("implement me") }
func (m *mockCmix) GetVerboseRounds() string                  { panic("implement me") }

////////////////////////////////////////////////////////////////////////////////
// Mock Connection Handler                                                    //
////////////////////////////////////////////////////////////////////////////////

func newMockListener(hearChan chan receive.Message) *mockListener {
	return &mockListener{hearChan: hearChan}
}

func (l *mockListener) Hear(item receive.Message) {
	l.hearChan <- item
}

func (l *mockListener) Name() string {
	return "mockListener"
}

type mockConnectionHandler struct {
	msgMap    map[catalog.MessageType][][]byte
	listeners map[catalog.MessageType]receive.Listener
	sync.Mutex
}

func newMockConnectionHandler() *mockConnectionHandler {
	return &mockConnectionHandler{
		msgMap:    make(map[catalog.MessageType][][]byte),
		listeners: make(map[catalog.MessageType]receive.Listener),
	}
}

// Tests that mockConnection adheres to the connection interface.
var _ connection = (*mockConnection)(nil)

type mockConnection struct {
	myID      *id.ID
	recipient *id.ID
	handler   *mockConnectionHandler
	t         *testing.T
}

type mockListener struct {
	hearChan chan receive.Message
}

func newMockConnection(myID, recipient *id.ID, handler *mockConnectionHandler,
	t *testing.T) *mockConnection {
	return &mockConnection{
		myID:      myID,
		recipient: recipient,
		handler:   handler,
		t:         t,
	}
}

func (m *mockConnection) GetPartner() partner.Manager {
	return partner.NewTestManager(m.recipient, nil, nil, m.t)
}

// SendE2E adds the message to the e2e handler map.
func (m *mockConnection) SendE2E(mt catalog.MessageType, payload []byte,
	_ e2e.Params) (cryptoE2e.SendReport, error) {
	m.handler.Lock()
	defer m.handler.Unlock()

	m.handler.listeners[mt].Hear(receive.Message{
		MessageType: mt,
		Payload:     payload,
		Sender:      m.myID,
	})

	return cryptoE2e.SendReport{RoundList: []id.Round{42}}, nil
}

func (m *mockConnection) RegisterListener(mt catalog.MessageType,
	listener receive.Listener) (receive.ListenerID, error) {
	m.handler.Lock()
	defer m.handler.Unlock()
	m.handler.listeners[mt] = listener
	return receive.ListenerID{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Mock Storage Session                                                       //
////////////////////////////////////////////////////////////////////////////////

type mockStorage struct {
	kv        *versioned.KV
	cmixGroup *cyclic.Group
}

func newMockStorage() *mockStorage {
	b := make([]byte, 768)
	rng := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG).GetStream()
	_, _ = rng.Read(b)
	rng.Close()

	return &mockStorage{
		kv:        versioned.NewKV(ekv.MakeMemstore()),
		cmixGroup: cyclic.NewGroup(large.NewIntFromBytes(b), large.NewInt(2)),
	}
}

func (m *mockStorage) GetClientVersion() version.Version     { panic("implement me") }
func (m *mockStorage) Get(string) (*versioned.Object, error) { panic("implement me") }
func (m *mockStorage) Set(string, *versioned.Object) error   { panic("implement me") }
func (m *mockStorage) Delete(string) error                   { panic("implement me") }
func (m *mockStorage) GetKV() *versioned.KV                  { return m.kv }
func (m *mockStorage) GetCmixGroup() *cyclic.Group           { return m.cmixGroup }
func (m *mockStorage) GetE2EGroup() *cyclic.Group            { panic("implement me") }
func (m *mockStorage) ForwardRegistrationStatus(storage.RegistrationStatus) error {
	panic("implement me")
}
func (m *mockStorage) GetRegistrationStatus() storage.RegistrationStatus      { panic("implement me") }
func (m *mockStorage) SetRegCode(string)                                      { panic("implement me") }
func (m *mockStorage) GetRegCode() (string, error)                            { panic("implement me") }
func (m *mockStorage) SetNDF(*ndf.NetworkDefinition)                          { panic("implement me") }
func (m *mockStorage) GetNDF() *ndf.NetworkDefinition                         { panic("implement me") }
func (m *mockStorage) GetTransmissionID() *id.ID                              { panic("implement me") }
func (m *mockStorage) GetTransmissionSalt() []byte                            { panic("implement me") }
func (m *mockStorage) GetReceptionID() *id.ID                                 { panic("implement me") }
func (m *mockStorage) GetReceptionSalt() []byte                               { panic("implement me") }
func (m *mockStorage) GetReceptionRSA() *rsa.PrivateKey                       { panic("implement me") }
func (m *mockStorage) GetTransmissionRSA() *rsa.PrivateKey                    { panic("implement me") }
func (m *mockStorage) IsPrecanned() bool                                      { panic("implement me") }
func (m *mockStorage) SetUsername(string) error                               { panic("implement me") }
func (m *mockStorage) GetUsername() (string, error)                           { panic("implement me") }
func (m *mockStorage) PortableUserInfo() user.Info                            { panic("implement me") }
func (m *mockStorage) GetTransmissionRegistrationValidationSignature() []byte { panic("implement me") }
func (m *mockStorage) GetReceptionRegistrationValidationSignature() []byte    { panic("implement me") }
func (m *mockStorage) GetRegistrationTimestamp() time.Time                    { panic("implement me") }
func (m *mockStorage) SetTransmissionRegistrationValidationSignature([]byte)  { panic("implement me") }
func (m *mockStorage) SetReceptionRegistrationValidationSignature([]byte)     { panic("implement me") }
func (m *mockStorage) SetRegistrationTimestamp(int64)                         { panic("implement me") }
