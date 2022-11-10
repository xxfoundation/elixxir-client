////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"github.com/cloudflare/circl/dh/sidh"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/gateway"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner"
	"gitlab.com/elixxir/client/v4/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage"
	userStorage "gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/client/v4/xxdk"
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
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Mock xxdk.E2e                                                              //
////////////////////////////////////////////////////////////////////////////////

type mockUser struct {
	rid xxdk.ReceptionIdentity
	c   cmix.Client
	e2e e2e.Handler
	s   storage.Session
	rng *fastRNG.StreamGenerator
}

func newMockUser(rid *id.ID, c cmix.Client, e2e e2e.Handler, s storage.Session,
	rng *fastRNG.StreamGenerator) *mockUser {
	return &mockUser{
		rid: xxdk.ReceptionIdentity{ID: rid},
		c:   c,
		e2e: e2e,
		s:   s,
		rng: rng,
	}
}

func (m *mockUser) GetStorage() storage.Session                  { return m.s }
func (m *mockUser) GetReceptionIdentity() xxdk.ReceptionIdentity { return m.rid }
func (m *mockUser) GetCmix() cmix.Client                         { return m.c }
func (m *mockUser) GetE2E() e2e.Handler                          { return m.e2e }
func (m *mockUser) GetRng() *fastRNG.StreamGenerator             { return m.rng }

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

func newMockCmix(myID *id.ID, handler *mockCmixHandler, storage *mockStorage) *mockCmix {
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
	[]byte, cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}

func (m *mockCmix) SendMany(messages []cmix.TargetedCmixMessage,
	_ cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
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
	return rounds.Round{ID: 42}, []ephemeral.Id{}, nil
}

func (m *mockCmix) SendWithAssembler(*id.ID, cmix.MessageAssembler,
	cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	panic("implement me")
}

func (m *mockCmix) AddIdentity(*id.ID, time.Time, bool)                       { panic("implement me") }
func (m *mockCmix) AddIdentityWithHistory(*id.ID, time.Time, time.Time, bool) { panic("implement me") }
func (m *mockCmix) RemoveIdentity(*id.ID)                                     { panic("implement me") }
func (m *mockCmix) GetIdentity(*id.ID) (identity.TrackedID, error)            { panic("implement me") }

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

func (m *mockCmix) DeleteClientFingerprints(*id.ID)                       { panic("implement me") }
func (m *mockCmix) AddService(*id.ID, message.Service, message.Processor) { panic("implement me") }
func (m *mockCmix) IncreaseParallelNodeRegistration(int) func() (stoppable.Stoppable, error) {
	panic("implement me")
}
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
	roundCallback cmix.RoundEventCallback, _ ...id.Round) {
	go roundCallback(true, false, map[id.Round]cmix.RoundResult{42: {}})
}

func (m *mockCmix) LookupHistoricalRound(id.Round, rounds.RoundResultCallback) error {
	panic("implement me")
}
func (m *mockCmix) SendToAny(func(host *connect.Host) (interface{}, error),
	*stoppable.Single) (interface{}, error) {
	panic("implement me")
}
func (m *mockCmix) SendToPreferred([]*id.ID, gateway.SendToPreferredFunc,
	*stoppable.Single, time.Duration) (interface{}, error) {
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

func (m *mockCmix) PauseNodeRegistrations(timeout time.Duration) error { return nil }
func (m *mockCmix) ChangeNumberOfNodeRegistrations(toRun int, timeout time.Duration) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Mock E2E Handler                                                           //
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

type mockE2eHandler struct {
	msgMap    map[id.ID]map[catalog.MessageType][][]byte
	listeners map[catalog.MessageType]receive.Listener
}

func newMockE2eHandler() *mockE2eHandler {
	return &mockE2eHandler{
		msgMap:    make(map[id.ID]map[catalog.MessageType][][]byte),
		listeners: make(map[catalog.MessageType]receive.Listener),
	}
}

type mockE2e struct {
	myID    *id.ID
	handler *mockE2eHandler
}

type mockListener struct {
	hearChan chan receive.Message
}

func newMockE2e(myID *id.ID, handler *mockE2eHandler) *mockE2e {
	return &mockE2e{
		myID:    myID,
		handler: handler,
	}
}

func (m *mockE2e) StartProcesses() (stoppable.Stoppable, error) { panic("implement me") }

// SendE2E adds the message to the e2e handler map.
func (m *mockE2e) SendE2E(mt catalog.MessageType, recipient *id.ID,
	payload []byte, _ e2e.Params) (cryptoE2e.SendReport, error) {

	m.handler.listeners[mt].Hear(receive.Message{
		MessageType: mt,
		Payload:     payload,
		Sender:      m.myID,
		RecipientID: recipient,
	})

	return cryptoE2e.SendReport{RoundList: []id.Round{42}}, nil
}

func (m *mockE2e) RegisterListener(_ *id.ID, mt catalog.MessageType,
	listener receive.Listener) receive.ListenerID {
	m.handler.listeners[mt] = listener
	return receive.ListenerID{}
}

func (m *mockE2e) RegisterFunc(string, *id.ID, catalog.MessageType, receive.ListenerFunc) receive.ListenerID {
	panic("implement me")
}
func (m *mockE2e) RegisterChannel(string, *id.ID, catalog.MessageType, chan receive.Message) receive.ListenerID {
	panic("implement me")
}
func (m *mockE2e) Unregister(receive.ListenerID)  { panic("implement me") }
func (m *mockE2e) UnregisterUserListeners(*id.ID) { panic("implement me") }
func (m *mockE2e) AddPartner(*id.ID, *cyclic.Int, *cyclic.Int, *sidh.PublicKey, *sidh.PrivateKey, session.Params, session.Params) (partner.Manager, error) {
	panic("implement me")
}
func (m *mockE2e) GetPartner(*id.ID) (partner.Manager, error)   { panic("implement me") }
func (m *mockE2e) DeletePartner(*id.ID) error                   { panic("implement me") }
func (m *mockE2e) DeletePartnerNotify(*id.ID, e2e.Params) error { panic("implement me") }
func (m *mockE2e) GetAllPartnerIDs() []*id.ID                   { panic("implement me") }
func (m *mockE2e) HasAuthenticatedChannel(*id.ID) bool          { panic("implement me") }
func (m *mockE2e) AddService(string, message.Processor) error   { panic("implement me") }
func (m *mockE2e) RemoveService(string) error                   { panic("implement me") }
func (m *mockE2e) SendUnsafe(catalog.MessageType, *id.ID, []byte, e2e.Params) ([]id.Round, time.Time, error) {
	panic("implement me")
}
func (m *mockE2e) EnableUnsafeReception()                    { panic("implement me") }
func (m *mockE2e) GetGroup() *cyclic.Group                   { panic("implement me") }
func (m *mockE2e) GetHistoricalDHPubkey() *cyclic.Int        { panic("implement me") }
func (m *mockE2e) GetHistoricalDHPrivkey() *cyclic.Int       { panic("implement me") }
func (m *mockE2e) GetReceptionID() *id.ID                    { panic("implement me") }
func (m *mockE2e) FirstPartitionSize() uint                  { panic("implement me") }
func (m *mockE2e) SecondPartitionSize() uint                 { panic("implement me") }
func (m *mockE2e) PartitionSize(uint) uint                   { panic("implement me") }
func (m *mockE2e) PayloadSize() uint                         { panic("implement me") }
func (m *mockE2e) RegisterCallbacks(e2e.Callbacks)           { panic("implement me") }
func (m *mockE2e) AddPartnerCallbacks(*id.ID, e2e.Callbacks) { panic("implement me") }
func (m *mockE2e) DeletePartnerCallbacks(*id.ID)             { panic("implement me") }

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
func (m *mockStorage) PortableUserInfo() userStorage.Info                     { panic("implement me") }
func (m *mockStorage) GetTransmissionRegistrationValidationSignature() []byte { panic("implement me") }
func (m *mockStorage) GetReceptionRegistrationValidationSignature() []byte    { panic("implement me") }
func (m *mockStorage) GetRegistrationTimestamp() time.Time                    { panic("implement me") }
func (m *mockStorage) SetTransmissionRegistrationValidationSignature([]byte)  { panic("implement me") }
func (m *mockStorage) SetReceptionRegistrationValidationSignature([]byte)     { panic("implement me") }
func (m *mockStorage) SetRegistrationTimestamp(int64)                         { panic("implement me") }
