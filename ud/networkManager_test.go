package ud

import (
	"bytes"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/gateway"
	"gitlab.com/elixxir/client/cmix/identity"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

// testNetworkManager is a mock implementation of the udCmix interface.
type testNetworkManager struct {
	requestProcess    message.Processor
	instance          *network.Instance
	testingFace       interface{}
	c                 contact.Contact
	responseProcessor message.Processor
}

func (tnm *testNetworkManager) SendWithAssembler(recipient *id.ID, assembler cmix.MessageAssembler,
	cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {

	msg := format.NewMessage(tnm.instance.GetE2EGroup().GetP().ByteLen())

	var rid id.Round = 123
	ephemeralId := new(ephemeral.Id)

	fingerprint, service, payload, mac, err := assembler(rid)
	if err != nil {
		return rid, *ephemeralId, err
	}

	// Build message. Will panic if inputs are not correct.
	msg.SetKeyFP(fingerprint)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetSIH(service.Hash(msg.GetContents()))
	// If the recipient for a call to Send is UD, then this
	// is the request pathway. Call the UD processor to simulate
	// the UD picking up the request
	if bytes.Equal(tnm.instance.GetFullNdf().
		Get().UDB.ID,
		recipient.Bytes()) {
		tnm.responseProcessor.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})

	} else {
		// This should happen when the mock UD service Sends back a response.
		// Calling process mocks up the requester picking up the response.
		tnm.requestProcess.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}

	return 0, ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) Send(recipient *id.ID, fingerprint format.Fingerprint,
	service message.Service,
	payload, mac []byte, cmixParams cmix.CMIXParams) (id.Round, ephemeral.Id, error) {
	msg := format.NewMessage(tnm.instance.GetE2EGroup().GetP().ByteLen())
	// Build message. Will panic if inputs are not correct.
	msg.SetKeyFP(fingerprint)
	msg.SetContents(payload)
	msg.SetMac(mac)
	msg.SetSIH(service.Hash(msg.GetContents()))
	// If the recipient for a call to Send is UD, then this
	// is the request pathway. Call the UD processor to simulate
	// the UD picking up the request
	if bytes.Equal(tnm.instance.GetFullNdf().
		Get().UDB.ID,
		recipient.Bytes()) {
		tnm.responseProcessor.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})

	} else {
		// This should happen when the mock UD service Sends back a response.
		// Calling process mocks up the requester picking up the response.
		tnm.requestProcess.Process(msg, receptionID.EphemeralIdentity{}, rounds.Round{})
	}

	return 0, ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) AddFingerprint(identity *id.ID,
	fingerprint format.Fingerprint, mp message.Processor) error {
	// AddFingerprint gets called in both the request and response
	// code-paths. We only want to set in the code-path transmitting
	// from UD
	if !bytes.Equal(tnm.instance.GetFullNdf().Get().UDB.ID,
		identity.Bytes()) {
		tnm.requestProcess = mp
	}

	return nil
}

func (tnm *testNetworkManager) AddService(clientID *id.ID,
	newService message.Service,
	response message.Processor) {
	tnm.responseProcessor = response
	return
}

func (tnm *testNetworkManager) CheckInProgressMessages() {
	return
}

func (tnm *testNetworkManager) GetMaxMessageLength() int {
	return 700
}

func (tnm *testNetworkManager) AddIdentity(id *id.ID, validUntil time.Time, persistent bool) {
	return
}

func (tnm *testNetworkManager) DeleteClientFingerprints(identity *id.ID) {
	return
}

func (tnm *testNetworkManager) Process(ecrMsg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

}

func (tnm *testNetworkManager) String() string {
	return "mockPRocessor"
}

func (tnm *testNetworkManager) DeleteService(clientID *id.ID, toDelete message.Service, processor message.Processor) {
	return
}

func (tnm *testNetworkManager) IsHealthy() bool {
	return true
}

func (tnm *testNetworkManager) GetAddressSpace() uint8 {
	return 8
}

func (tnm *testNetworkManager) GetInstance() *network.Instance {
	return tnm.instance
}

func (tnm *testNetworkManager) Follow(report cmix.ClientErrorReport) (stoppable.Stoppable, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetVerboseRounds() string {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToAny(sendFunc func(host *connect.Host) (interface{}, error), stop *stoppable.Single) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendMany(messages []cmix.TargetedCmixMessage, p cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveIdentity(id *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetIdentity(get *id.ID) (identity.TrackedID, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteFingerprint(identity *id.ID, fingerprint format.Fingerprint) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) DeleteClientService(clientID *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TrackServices(tracker message.ServicesTracker) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) WasHealthy() bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) AddHealthCallback(f func(bool)) uint64 {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RemoveHealthCallback(u uint64) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) HasNode(nid *id.ID) bool {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) NumRegisteredNodes() int {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) TriggerNodeRegistration(nid *id.ID) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetRoundResults(timeout time.Duration, roundCallback cmix.RoundEventCallback, roundList ...id.Round) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) LookupHistoricalRound(rid id.Round, callback rounds.RoundResultCallback) error {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SendToPreferred(targets []*id.ID, sendFunc gateway.SendToPreferredFunc, stop *stoppable.Single, timeout time.Duration) (interface{}, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) SetGatewayFilter(f gateway.Filter) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) GetHostParams() connect.HostParams {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) RegisterAddressSpaceNotification(tag string) (chan uint8, error) {
	//TODO implement me
	panic("implement me")
}

func (tnm *testNetworkManager) UnregisterAddressSpaceNotification(tag string) {
	//TODO implement me
	panic("implement me")
}
