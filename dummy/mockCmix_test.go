////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

// mockCmix is a testing structure that adheres to cmix.Client.
type mockCmix struct {
	messages map[id.ID]format.Message
	sync.RWMutex
	payloadSize int
}

func newMockCmix(payloadSize int) cmix.Client {

	return &mockCmix{
		messages:    make(map[id.ID]format.Message),
		payloadSize: payloadSize,
	}
}

func (m *mockCmix) Send(recipient *id.ID, fingerprint format.Fingerprint, service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.Lock()
	defer m.Unlock()
	m.messages[*recipient] = generateMessage(m.payloadSize, fingerprint, service, payload, mac)

	return rounds.Round{}, ephemeral.Id{}, nil
}

func (m *mockCmix) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	m.Lock()
	defer m.Unlock()

	fingerprint, service, payload, mac, err := assembler(42)
	if err != nil {
		return rounds.Round{}, ephemeral.Id{}, err
	}
	m.messages[*recipient] = generateMessage(m.payloadSize, fingerprint, service, payload, mac)

	return rounds.Round{}, ephemeral.Id{}, nil
}

func (m *mockCmix) GetMsgListLen() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.messages)
}

func (m *mockCmix) GetMsgList() map[id.ID]format.Message {
	m.RLock()
	defer m.RUnlock()
	return m.messages
}

func (m mockCmix) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetMaxMessageLength() int {
	return 100
}

func (m *mockCmix) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCmix) AddIdentityWithHistory(id *id.ID, validUntil, beginning time.Time, persistent bool) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCmix) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	//TODO implement me
	panic("implement me")
}

func (m *mockCmix) RemoveIdentity(id *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint, mp message.Processor) error {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) DeleteClientFingerprints(identity *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) AddService(clientID *id.ID, newService message.Service, response message.Processor) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) IncreaseParallelNodeRegistration(int) func() (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}


func (m mockCmix) DeleteService(clientID *id.ID, toDelete message.Service, processor message.Processor) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) DeleteClientService(clientID *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) TrackServices(tracker message.ServicesTracker) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) CheckInProgressMessages() {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) IsHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) WasHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) AddHealthCallback(f func(bool)) uint64 {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) RemoveHealthCallback(u uint64) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) HasNode(nid *id.ID) bool {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) NumRegisteredNodes() int {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) TriggerNodeRegistration(nid *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) SetGatewayFilter(f gateway.Filter) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetHostParams() connect.HostParams {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetAddressSpace() uint8 {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) UnregisterAddressSpaceNotification(tag string) {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetInstance() *network.Instance {
	//TODO implement me
	panic("implement me")
}

func (m mockCmix) GetVerboseRounds() string {
	//TODO implement me
	panic("implement me")
}
