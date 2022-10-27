////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

// testNetworkManager is a test implementation of NetworkManager interface.
type testNetworkManager struct {
	receptionMessages [][]format.Message
	sendMessages      [][]cmix.TargetedCmixMessage
	errSkip           int
	sendErr           int
	grp               *cyclic.Group
	sync.RWMutex
}

func newTestNetworkManager(sendErr int) cmix.Client {
	return &testNetworkManager{
		receptionMessages: [][]format.Message{},
		sendMessages:      [][]cmix.TargetedCmixMessage{},
		grp:               getGroup(),
		sendErr:           sendErr,
	}
}

func (tnm *testNetworkManager) SendMany(messages []cmix.TargetedCmixMessage, _ cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	if tnm.sendErr == 1 {
		return rounds.Round{}, nil, errors.New("SendManyCMIX error")
	}

	tnm.Lock()
	defer tnm.Unlock()

	tnm.sendMessages = append(tnm.sendMessages, messages)

	var receiveMessages []format.Message
	for _, msg := range messages {
		receiveMsg := format.NewMessage(tnm.grp.GetP().ByteLen())
		receiveMsg.SetMac(msg.Mac)
		receiveMsg.SetContents(msg.Payload)
		receiveMsg.SetKeyFP(msg.Fingerprint)
		receiveMessages = append(receiveMessages, receiveMsg)
	}
	tnm.receptionMessages = append(tnm.receptionMessages, receiveMessages)
	return rounds.Round{}, nil, nil
}

func (*testNetworkManager) AddService(*id.ID, message.Service, message.Processor) {}
func (*testNetworkManager) IncreaseParallelNodeRegistration(int) func() (stoppable.Stoppable, error) {
	return nil
}
func (*testNetworkManager) DeleteService(*id.ID, message.Service, message.Processor) {}

/////////////////////////////////////////////////////////////////////////////////////
// Unused & unimplemented methods of the test object ////////////////////////////////
/////////////////////////////////////////////////////////////////////////////////////

func (tnm *testNetworkManager) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendWithAssembler(recipient *id.ID,
	assembler cmix.MessageAssembler, cmixParams cmix.CMIXParams) (rounds.Round,
	ephemeral.Id, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) Send(recipient *id.ID, fingerprint format.Fingerprint, service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddIdentityWithHistory(id *id.ID, validUntil, beginning time.Time, persistent bool) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveIdentity(id *id.ID) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddFingerprint(identity *id.ID, fingerprint format.Fingerprint, mp message.Processor) error {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteClientFingerprints(identity *id.ID) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteClientService(clientID *id.ID) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TrackServices(tracker message.ServicesTracker) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) CheckInProgressMessages() {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) IsHealthy() bool {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) WasHealthy() bool {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddHealthCallback(f func(bool)) uint64 {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveHealthCallback(u uint64) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) HasNode(nid *id.ID) bool {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) NumRegisteredNodes() int {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TriggerNodeRegistration(nid *id.ID) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SetGatewayFilter(f gateway.Filter) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetHostParams() connect.HostParams {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetAddressSpace() uint8 {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) UnregisterAddressSpaceNotification(tag string) {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetInstance() *network.Instance {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetVerboseRounds() string {
	// TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetMaxMessageLength() int {
	return format.NewMessage(tnm.grp.GetP().ByteLen()).ContentsSize()
}
