////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package message

import (
	"testing"
	"time"

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/sih"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
)

type testProcessor struct {
}

func (t *testProcessor) Process(message format.Message, receptionID receptionID.EphemeralIdentity, round rounds.Round) {

}

func (t *testProcessor) String() string {
	return ""
}

// Test that a message is handled when no service or fingerprint exists and message needs to be removed
func Test_handler_handleMessage_NoResult(t *testing.T) {
	garbled, err := NewOrLoadMeteredCmixMessageBuffer(versioned.NewKV(ekv.MakeMemstore()), inProcessKey)
	if err != nil {
		t.Errorf("Failed to load or new the Garbled Messages system: %v", err)
	}
	m := handler{
		events:    event.NewEventManager(),
		param:     GetDefaultParams(),
		inProcess: garbled,
	}
	testId := id.NewIdFromUInt(2, id.User, t)
	m.ServicesManager = *NewServices()

	contents := []byte{4, 4}
	lazyPreimage := sih.MakePreimage(contents, "test")
	ecrMsg := format.NewMessage(2056)
	ecrMsg.SetContents(contents)
	ecrMsg.SetSIH(sih.Hash(lazyPreimage, ecrMsg.GetContents()))

	testRound := rounds.Round{
		Timestamps:       make(map[states.Round]time.Time),
		AddressSpaceSize: 18,
		Raw:              &pb.RoundInfo{Timestamps: make([]uint64, states.NUM_STATES)},
	}
	testRound.Timestamps[states.QUEUED] = time.Now()
	testRound.Raw.Timestamps[states.QUEUED] = uint64(time.Now().Unix())

	ephId := receptionID.BuildIdentityFromRound(testId, testRound)
	bundle := Bundle{
		Identity:  ephId,
		RoundInfo: testRound,
	}

	m.inProcess.Add(ecrMsg, bundle.RoundInfo.Raw, bundle.Identity)
	m.handleMessage(m.param.MaxChecksInProcessMessage, time.Now().Add(-2*m.param.InProcessMessageWait),
		ecrMsg, bundle)
	if len(m.inProcess.mb.GetMessages()) != 0 {
		t.Errorf("Expected to remove message from inProgress!")
	}
}

// Test that a message is handled correctly via fingerprinting
func Test_handler_handleMessageHelper(t *testing.T) {
	m := handler{
		events: event.NewEventManager(),
	}
	testId := id.NewIdFromUInt(2, id.User, t)
	m.FingerprintsManager = *newFingerprints(testId)

	ecrMsg := format.NewMessage(2056)
	fp := format.NewFingerprint([]byte{0, 2})
	ecrMsg.SetKeyFP(fp)

	testRound := rounds.Round{
		Timestamps:       make(map[states.Round]time.Time),
		AddressSpaceSize: 18,
	}
	testRound.Timestamps[states.QUEUED] = time.Now()

	ephId := receptionID.BuildIdentityFromRound(testId, testRound)
	bundle := Bundle{
		Identity:  ephId,
		RoundInfo: testRound,
	}

	err := m.AddFingerprint(testId, fp, &testProcessor{})
	if err != nil {
		t.Errorf("Unexpected failure to AddFignerprint: %+v", err)
	}
	result := m.handleMessageHelper(ecrMsg, bundle)
	if !result {
		t.Errorf("Expected handleMessage success!")
	}
}

// Test that a message is handled when no service or fingerprint exists
func Test_handler_handleMessageHelper_NoResult(t *testing.T) {
	m := handler{
		events: event.NewEventManager(),
	}
	testId := id.NewIdFromUInt(2, id.User, t)
	m.ServicesManager = *NewServices()

	contents := []byte{4, 4}
	lazyPreimage := sih.MakePreimage(contents, "test")
	ecrMsg := format.NewMessage(2056)
	ecrMsg.SetContents(contents)
	ecrMsg.SetSIH(sih.Hash(lazyPreimage, ecrMsg.GetContents()))

	testRound := rounds.Round{
		Timestamps:       make(map[states.Round]time.Time),
		AddressSpaceSize: 18,
	}
	testRound.Timestamps[states.QUEUED] = time.Now()

	ephId := receptionID.BuildIdentityFromRound(testId, testRound)
	bundle := Bundle{
		Identity:  ephId,
		RoundInfo: testRound,
	}

	result := m.handleMessageHelper(ecrMsg, bundle)
	if result {
		t.Errorf("Expected handleMessage failure!")
	}
}

// Test that a message is handled correctly via services
func Test_handler_handleMessageHelper_Service(t *testing.T) {
	m := handler{
		events: event.NewEventManager(),
	}
	testId := id.NewIdFromUInt(2, id.User, t)
	m.ServicesManager = *NewServices()

	contents := []byte{4, 4}
	lazyPreimage := sih.MakePreimage(contents, "test")

	s := Service{
		Identifier:   nil,
		Tag:          "test",
		lazyPreimage: &lazyPreimage,
	}

	ecrMsg := format.NewMessage(2056)
	ecrMsg.SetContents(contents)
	ecrMsg.SetSIH(s.Hash(ecrMsg.GetContents()))

	testRound := rounds.Round{
		Timestamps:       make(map[states.Round]time.Time),
		AddressSpaceSize: 18,
	}
	testRound.Timestamps[states.QUEUED] = time.Now()

	ephId := receptionID.BuildIdentityFromRound(testId, testRound)
	bundle := Bundle{
		Identity:  ephId,
		RoundInfo: testRound,
	}

	processor := &testProcessor{}

	m.AddService(testId, s, processor)
	result := m.handleMessageHelper(ecrMsg, bundle)
	if !result {
		t.Errorf("Expected handleMessage success!")
	}
}
