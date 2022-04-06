///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package single

import (
	"bytes"
	"errors"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	ephemeral2 "gitlab.com/elixxir/client/network/identity/receptionID"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/network"
	contact2 "gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"
)

// Happy path.
func Test_newManager(t *testing.T) {
	client := &api.Client{}
	e := &Manager{
		client: client,
		p:      newPending(),
	}
	m := newManager(client, &ephemeral2.Store{})

	if e.client != m.client || e.store != m.store || e.net != m.net ||
		e.rng != m.rng || !reflect.DeepEqual(e.p, m.p) {
		t.Errorf("NewHandler() did not return the expected new State."+
			"\nexpected: %+v\nreceived: %+v", e, m)
	}
}

// Happy path.
func TestManager_StartProcesses(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "Test tag"
	payload := make([]byte, 130)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	transmitMsg, _, rid, _, err := m.makeTransmitCmixMessage(partner, payload,
		tag, 8, 32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	_, sender, err := m.processTransmission(transmitMsg, singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey()))

	replyMsgs, err := m.makeReplyCmixMessages(sender, payload)
	if err != nil {
		t.Fatalf("Failed to generate reply CMIX messages: %+v", err)
	}

	receiveMsg := message.Receive{
		Payload:     transmitMsg.Marshal(),
		MessageType: message.Raw,
		Sender:      rid,
		RecipientID: partner.ID,
	}

	m.callbackMap.registerCallback(tag, callback)

	_, _ = m.StartProcesses()
	m.swb.(*switchboard.Switchboard).Speak(receiveMsg)

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		if !bytes.Equal(results.payload, payload) {
			t.Errorf("Callback received wrong payload."+
				"\nexpected: %+v\nreceived: %+v", payload, results.payload)
		}
	case <-timer.C:
		t.Errorf("Callback failed to be called.")
	}

	callbackFunc, callbackFuncChan := createReplyComm()
	m.p.Lock()
	m.p.singleUse[*rid] = newState(sender.dhKey, sender.maxParts, callbackFunc)
	m.p.Unlock()
	eid, _, _, _ := ephemeral.GetId(partner.ID, id.ArrIDLen, netTime.Now().UnixNano())
	replyMsg := message.Receive{
		Payload:     replyMsgs[0].Marshal(),
		MessageType: message.Raw,
		Sender:      partner.ID,
		RecipientID: rid,
		EphemeralID: eid,
	}

	go func() {
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-timer.C:
			t.Errorf("quitChan never set.")
		case <-m.p.singleUse[*rid].quitChan:
		}
	}()

	m.swb.(*switchboard.Switchboard).Speak(replyMsg)

	timer = time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackFuncChan:
		if !bytes.Equal(results.payload, payload) {
			t.Errorf("Callback received wrong payload."+
				"\nexpected: %+v\nreceived: %+v", payload, results.payload)
		}
	case <-timer.C:
		t.Errorf("Callback failed to be called.")
	}
}

// Happy path: tests that the stoppable stops both routines.
func TestManager_StartProcesses_Stop(t *testing.T) {
	m := newTestManager(0, false, t)
	partner := contact2.Contact{
		ID:       id.NewIdFromString("recipientID", id.User, t),
		DhPubKey: m.store.E2e().GetDHPublicKey(),
	}
	tag := "Test tag"
	payload := make([]byte, 130)
	rand.New(rand.NewSource(42)).Read(payload)
	callback, callbackChan := createReceiveComm()

	transmitMsg, _, rid, _, err := m.makeTransmitCmixMessage(partner, payload,
		tag, 8, 32, 30*time.Second, netTime.Now(), rand.New(rand.NewSource(42)))
	if err != nil {
		t.Fatalf("Failed to create tranmission CMIX message: %+v", err)
	}

	_, sender, err := m.processTransmission(transmitMsg, singleUse.NewTransmitFingerprint(m.store.E2e().GetDHPublicKey()))

	replyMsgs, err := m.makeReplyCmixMessages(sender, payload)
	if err != nil {
		t.Fatalf("Failed to generate reply CMIX messages: %+v", err)
	}

	receiveMsg := message.Receive{
		Payload:     transmitMsg.Marshal(),
		MessageType: message.Raw,
		Sender:      rid,
		RecipientID: partner.ID,
	}

	m.callbackMap.registerCallback(tag, callback)

	stop, _ := m.StartProcesses()
	if !stop.IsRunning() {
		t.Error("Stoppable is not running.")
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close: %+v", err)
	}

	// Wait for the stoppable to close
	for !stop.IsStopped() {
		time.Sleep(10 * time.Millisecond)
	}

	m.swb.(*switchboard.Switchboard).Speak(receiveMsg)

	timer := time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackChan:
		t.Errorf("Callback called when the thread should have stopped."+
			"\npayload: %+v\ncontact: %+v", results.payload, results.c)
	case <-timer.C:
	}

	callbackFunc, callbackFuncChan := createReplyComm()
	m.p.Lock()
	m.p.singleUse[*rid] = newState(sender.dhKey, sender.maxParts, callbackFunc)
	m.p.Unlock()
	eid, _, _, _ := ephemeral.GetId(partner.ID, id.ArrIDLen, netTime.Now().UnixNano())
	replyMsg := message.Receive{
		Payload:     replyMsgs[0].Marshal(),
		MessageType: message.Raw,
		Sender:      partner.ID,
		RecipientID: rid,
		EphemeralID: eid,
	}

	m.swb.(*switchboard.Switchboard).Speak(replyMsg)

	timer = time.NewTimer(50 * time.Millisecond)

	select {
	case results := <-callbackFuncChan:
		t.Errorf("Callback called when the thread should have stopped."+
			"\npayload: %+v\nerror: %+v", results.payload, results.err)
	case <-timer.C:
	}
}

type receiveCommData struct {
	payload []byte
	c       Contact
}

func createReceiveComm() (ReceiveComm, chan receiveCommData) {
	callbackChan := make(chan receiveCommData)

	callback := func(payload []byte, c Contact) {
		callbackChan <- receiveCommData{payload: payload, c: c}
	}
	return callback, callbackChan
}

func newTestManager(timeout time.Duration, cmixErr bool, t *testing.T) *Manager {
	return &Manager{
		client:      nil,
		store:       storage.InitTestingSession(t),
		reception:   ephemeral2.NewStore(versioned.NewKV(make(ekv.Memstore))),
		swb:         switchboard.New(),
		net:         newTestNetworkManager(timeout, cmixErr, t),
		rng:         fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		p:           newPending(),
		callbackMap: newCallbackMap(),
	}
}

func newTestNetworkManager(timeout time.Duration, cmixErr bool, t *testing.T) interfaces.NetworkManager {
	instanceComms := &connect.ProtoComms{
		Manager: connect.NewManagerTesting(t),
	}

	thisInstance, err := network.NewInstanceTesting(instanceComms, getNDF(),
		getNDF(), nil, nil, t)
	if err != nil {
		t.Fatalf("Failed to create new test instance: %v", err)
	}

	return &testNetworkManager{
		instance:    thisInstance,
		msgs:        []format.Message{},
		cmixTimeout: timeout,
		cmixErr:     cmixErr,
	}
}

// testNetworkManager is a test implementation of NetworkManager interface.
type testNetworkManager struct {
	instance    *network.Instance
	msgs        []format.Message
	cmixTimeout time.Duration
	cmixErr     bool
	sync.RWMutex
}

func (tnm *testNetworkManager) GetMsg(i int) format.Message {
	tnm.RLock()
	defer tnm.RUnlock()
	return tnm.msgs[i]
}

func (tnm *testNetworkManager) SendE2E(message.Send, params.E2E, *stoppable.Single) ([]id.Round, e2e.MessageID, time.Time, error) {
	return nil, e2e.MessageID{}, time.Time{}, nil
}

func (tnm *testNetworkManager) GetVerboseRounds() string {
	return ""
}

func (tnm *testNetworkManager) SendUnsafe(_ message.Send, _ params.Unsafe) ([]id.Round, error) {
	return []id.Round{}, nil
}

func (tnm *testNetworkManager) SendCMIX(msg format.Message, _ *id.ID, _ params.CMIX) (id.Round, ephemeral.Id, error) {
	if tnm.cmixTimeout != 0 {
		time.Sleep(tnm.cmixTimeout)
	} else if tnm.cmixErr {
		return 0, ephemeral.Id{}, errors.New("sendCMIX error")
	}

	tnm.Lock()
	defer tnm.Unlock()

	tnm.msgs = append(tnm.msgs, msg)

	return id.Round(rand.Uint64()), ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) SendManyCMIX(messages []message.TargetedCmixMessage, p params.CMIX) (id.Round, []ephemeral.Id, error) {
	if tnm.cmixTimeout != 0 {
		time.Sleep(tnm.cmixTimeout)
	} else if tnm.cmixErr {
		return 0, []ephemeral.Id{}, errors.New("sendCMIX error")
	}

	tnm.Lock()
	defer tnm.Unlock()

	for _, msg := range messages {
		tnm.msgs = append(tnm.msgs, msg.Message)
	}

	return id.Round(rand.Uint64()), []ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) GetInstance() *network.Instance {
	return tnm.instance
}

type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(p int, a, b, c string) {}
func (t *testNetworkManager) GetEventManager() event.Manager {
	return &dummyEventMgr{}
}

func (tnm *testNetworkManager) GetHealthTracker() interfaces.HealthTracker {
	return nil
}

func (tnm *testNetworkManager) Follow(report interfaces.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}

func (tnm *testNetworkManager) CheckGarbledMessages() {}

func (tnm *testNetworkManager) InProgressRegistrations() int {
	return 0
}

func (tnm *testNetworkManager) GetSender() *gateway.Sender {
	return nil
}

func (tnm *testNetworkManager) GetAddressSize() uint8 { return 16 }

func (tnm *testNetworkManager) RegisterAddressSizeNotification(string) (chan uint8, error) {
	return nil, nil
}

func (tnm *testNetworkManager) UnregisterAddressSizeNotification(string) {}
func (tnm *testNetworkManager) SetPoolFilter(gateway.Filter)             {}

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B" +
				"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE" +
				"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F" +
				"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041" +
				"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45" +
				"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209" +
				"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29" +
				"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E" +
				"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2" +
				"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696" +
				"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E" +
				"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873" +
				"847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
				"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
				"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
				"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
				"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
				"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
				"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
				"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
				"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
				"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
				"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
				"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
				"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
				"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
				"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}
