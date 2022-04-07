package cmix

import (
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

/*
type dummyEvent struct{}

func (e *dummyEvent) Report(priority int, category, evtType, details string) {}

// Unit test
func Test_attemptSendCmix(t *testing.T) {
	sess1 := storage.InitTestingSession(t)

	sess2 := storage.InitTestingSession(t)

	events := &dummyEvent{}

	sw := switchboard.New()
	l := message2.TestListener{
		ch: make(chan bool),
	}
	sw.RegisterListener(sess2.GetUser().TransmissionID, message.Raw, l)
	comms, err := client.NewClientComms(sess1.GetUser().TransmissionID, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}
	inst, err := network.NewInstanceTesting(comms.ProtoComms, message2.getNDF(), nil, nil, nil, t)
	if err != nil {
		t.Errorf("Failed to start instance: %+v", err)
	}
	now := netTime.Now()
	nid1 := id.NewIdFromString("zezima", id.Node, t)
	nid2 := id.NewIdFromString("jakexx360", id.Node, t)
	nid3 := id.NewIdFromString("westparkhome", id.Node, t)
	grp := cyclic.NewGroup(large.NewInt(7), large.NewInt(13))
	sess1.Cmix().Add(nid1, grp.NewInt(1), 0, nil)
	sess1.Cmix().Add(nid2, grp.NewInt(2), 0, nil)
	sess1.Cmix().Add(nid3, grp.NewInt(3), 0, nil)

	timestamps := []uint64{
		uint64(now.Add(-30 * time.Second).UnixNano()), //PENDING
		uint64(now.Add(-25 * time.Second).UnixNano()), //PRECOMPUTING
		uint64(now.Add(-5 * time.Second).UnixNano()),  //STANDBY
		uint64(now.Add(5 * time.Second).UnixNano()),   //QUEUED
		0} //REALTIME

	ri := &mixmessages.RoundInfo{
		ID:                         3,
		UpdateID:                   0,
		State:                      uint32(states.QUEUED),
		BatchSize:                  0,
		Topology:                   [][]byte{nid1.Marshal(), nid2.Marshal(), nid3.Marshal()},
		Timestamps:                 timestamps,
		Errors:                     nil,
		ClientErrors:               nil,
		ResourceQueueTimeoutMillis: 0,
		Signature:                  nil,
		AddressSpaceSize:           4,
	}

	if err = testutils.SignRoundInfoRsa(ri, t); err != nil {
		t.Errorf("Failed to sign mock round info: %v", err)
	}

	pubKey, err := testutils.LoadPublicKeyTesting(t)
	if err != nil {
		t.Errorf("Failed to load a key for testing: %v", err)
	}
	rnd := ds.NewRound(ri, pubKey, nil)
	inst.GetWaitingRounds().Insert([]*ds.Round{rnd}, nil)
	i := internal.Internal{
		Session:          sess1,
		Switchboard:      sw,
		Rng:              fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		Comms:            comms,
		Health:           nil,
		TransmissionID:   sess1.GetUser().TransmissionID,
		Instance:         inst,
		NodeRegistration: nil,
	}
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(p, i.Rng, message2.getNDF(), &message2.MockSendCMIXComms{t: t}, i.Session, nil)
	if err != nil {
		t.Errorf("%+v", errors.New(err.Error()))
		return
	}
	m := message2.NewHandler(i, params.Network{Messages: params.Messages{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksRetryMessage:          20,
		RetryMessageWait:               time.Hour,
	}}, nil, sender)
	msgCmix := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())
	msgCmix.SetContents([]byte("test"))
	e2e.SetUnencrypted(msgCmix, m.Session.User().GetCryptographicIdentity().GetTransmissionID())
	_, _, err = sendCmixHelper(sender, msgCmix, sess2.GetUser().ReceptionID,
		params.GetDefaultCMIX(), make(map[string]interface{}), m.Instance, m.Session, m.nodeRegistration,
		m.Rng, events, m.TransmissionID, &message2.MockSendCMIXComms{t: t}, nil)
	if err != nil {
		t.Errorf("Failed to sendcmix: %+v", err)
		panic("t")
		return
	}
}*/

func TestManager_SendCMIX(t *testing.T) {
	m, err := newTestManager(t)
	if err != nil {
		t.Fatalf("Failed to create test client: %+v", err)
	}

	recipientID := id.NewIdFromString("zezima", id.User, t)
	contents := []byte("message")
	fp := format.NewFingerprint(contents)
	service := message.GetDefaultService(recipientID)
	mac := make([]byte, 32)
	_, err = csprng.NewSystemRNG().Read(mac)
	if err != nil {
		t.Errorf("Failed to read random mac bytes: %+v", err)
	}
	mac[0] = 0
	params := GetDefaultCMIXParams()
	rid, eid, err := m.Send(recipientID, fp, service, contents, mac, params)
	if err != nil {
		t.Errorf("Failed to sendcmix: %+v", err)
		t.FailNow()
	}
	t.Logf("Test of Send returned:\n\trid: %v\teid: %+v", rid, eid)
}
