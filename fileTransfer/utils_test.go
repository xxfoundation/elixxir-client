////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"bytes"
	"encoding/binary"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	ftStorage "gitlab.com/elixxir/client/storage/fileTransfer"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/fastRNG"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
	"io"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

// newFile generates a file with random data of size numParts * partSize.
// Returns the full file and the file parts. If the partSize allows, each part
// starts with a "|<[PART_001]" and ends with a ">|".
func newFile(numParts uint16, partSize int, prng io.Reader, t *testing.T) (
	[]byte, [][]byte) {
	const (
		prefix = "|<[PART_%3d]"
		suffix = ">|"
	)
	// Create file buffer of the expected size
	fileBuff := bytes.NewBuffer(make([]byte, 0, int(numParts)*partSize))
	partList := make([][]byte, numParts)

	// Create new rand.Rand with the seed generated from the io.Reader
	b := make([]byte, 8)
	_, err := prng.Read(b)
	if err != nil {
		t.Errorf("Failed to generate random seed: %+v", err)
	}
	seed := binary.LittleEndian.Uint64(b)
	randPrng := rand.New(rand.NewSource(int64(seed)))

	for partNum := range partList {
		s := RandStringBytes(partSize, randPrng)
		if len(s) >= (len(prefix) + len(suffix)) {
			partList[partNum] = []byte(
				prefix + s[:len(s)-(len(prefix)+len(suffix))] + suffix)
		} else {
			partList[partNum] = []byte(s)
		}

		fileBuff.Write(partList[partNum])
	}

	return fileBuff.Bytes(), partList
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// RandStringBytes generates a random string of length n consisting of the
// characters in letterBytes.
func RandStringBytes(n int, prng *rand.Rand) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[prng.Intn(len(letterBytes))]
	}
	return string(b)
}

// checkReceivedProgress compares the output of ReceivedTransfer.GetProgress to
// expected values.
func checkReceivedProgress(completed bool, received, total uint16,
	eCompleted bool, eReceived, eTotal uint16) error {
	if eCompleted != completed || eReceived != received || eTotal != total {
		return errors.Errorf("Returned progress does not match expected."+
			"\n          completed  received  total"+
			"\nexpected:     %5t       %3d    %3d"+
			"\nreceived:     %5t       %3d    %3d",
			eCompleted, eReceived, eTotal,
			completed, received, total)
	}

	return nil
}

// checkSentProgress compares the output of SentTransfer.GetProgress to expected
// values.
func checkSentProgress(completed bool, sent, arrived, total uint16,
	eCompleted bool, eSent, eArrived, eTotal uint16) error {
	if eCompleted != completed || eSent != sent || eArrived != arrived ||
		eTotal != total {
		return errors.Errorf("Returned progress does not match expected."+
			"\n          completed  sent  arrived  total"+
			"\nexpected:     %5t   %3d      %3d    %3d"+
			"\nreceived:     %5t   %3d      %3d    %3d",
			eCompleted, eSent, eArrived, eTotal,
			completed, sent, arrived, total)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// PRNG                                                                       //
////////////////////////////////////////////////////////////////////////////////

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }

// PrngErr is a PRNG that satisfies the csprng.Source interface. However, it
// always returns an error
type PrngErr struct{}

func NewPrngErr() csprng.Source             { return &PrngErr{} }
func (s *PrngErr) Read([]byte) (int, error) { return 0, errors.New("ReadFailure") }
func (s *PrngErr) SetSeed([]byte) error     { return errors.New("SetSeedFailure") }

////////////////////////////////////////////////////////////////////////////////
// Test Managers                                                              //
////////////////////////////////////////////////////////////////////////////////

// newTestManager creates a new Manager that has groups stored for testing. One
// of the groups in the list is also returned.
func newTestManager(sendErr bool, sendChan, sendE2eChan chan message.Receive,
	receiveCB interfaces.ReceiveCallback, kv *versioned.KV, t *testing.T) *Manager {

	if kv == nil {
		kv = versioned.NewKV(make(ekv.Memstore))
	}
	sent, err := ftStorage.NewSentFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to createw new SentFileTransfersStore: %+v", err)
	}
	received, err := ftStorage.NewReceivedFileTransfersStore(kv)
	if err != nil {
		t.Fatalf("Failed to createw new ReceivedFileTransfersStore: %+v", err)
	}

	net := newTestNetworkManager(sendErr, sendChan, sendE2eChan, t)

	// Returns an error on function and round failure on callback if sendErr is
	// set; otherwise, it reports round successes and returns nil
	rr := func(rIDs []id.Round, _ time.Duration, cb api.RoundEventCallback) error {
		rounds := make(map[id.Round]api.RoundResult, len(rIDs))
		for _, rid := range rIDs {
			if sendErr {
				rounds[rid] = api.Failed
			} else {
				rounds[rid] = api.Succeeded
			}
		}
		cb(!sendErr, false, rounds)
		if sendErr {
			return errors.New("SendError")
		}

		return nil
	}

	p := DefaultParams()
	avgNumMessages := (minPartsSendPerRound + maxPartsSendPerRound) / 2
	avgSendSize := avgNumMessages * (8192 / 8)
	p.MaxThroughput = int(time.Second) * avgSendSize

	oldTransfersRecovered := uint32(0)

	m := &Manager{
		receiveCB:             receiveCB,
		sent:                  sent,
		received:              received,
		sendQueue:             make(chan queuedPart, sendQueueBuffLen),
		oldTransfersRecovered: &oldTransfersRecovered,
		p:                     p,
		store:                 storage.InitTestingSession(t),
		swb:                   switchboard.New(),
		net:                   net,
		rng:                   fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		getRoundResults:       rr,
	}

	return m
}

// newTestManagerWithTransfers creates a new test manager with transfers added
// to it.
func newTestManagerWithTransfers(numParts []uint16, sendErr, addPartners bool,
	sendE2eChan chan message.Receive, receiveCB interfaces.ReceiveCallback,
	kv *versioned.KV, t *testing.T) (*Manager, []sentTransferInfo,
	[]receivedTransferInfo) {
	m := newTestManager(sendErr, sendE2eChan, nil, receiveCB, kv, t)
	sti := make([]sentTransferInfo, len(numParts))
	rti := make([]receivedTransferInfo, len(numParts))
	var err error

	partSize, err := m.getPartSize()
	if err != nil {
		t.Errorf("Failed to get part size: %+v", err)
	}

	// Add sent transfers to manager and populate the sentTransferInfo list
	for i := range sti {
		// Generate PRNG, the file and its parts, and the transfer key
		prng := NewPrng(int64(42 + i))
		file, parts := newFile(numParts[i], partSize, prng, t)
		key, _ := ftCrypto.NewTransferKey(prng)
		recipient := id.NewIdFromString("recipient"+strconv.Itoa(i), id.User, t)

		// Create a sentTransferInfo with all the transfer information
		sti[i] = sentTransferInfo{
			recipient: recipient,
			key:       key,
			parts:     parts,
			file:      file,
			numParts:  numParts[i],
			numFps:    calcNumberOfFingerprints(numParts[i], 0.5),
			retry:     0.5,
			period:    time.Millisecond,
			prng:      prng,
		}

		// Create sent progress callback and channel
		cbChan := make(chan sentProgressResults, 8)
		cb := func(completed bool, sent, arrived, total uint16,
			tr interfaces.FilePartTracker, err error) {
			cbChan <- sentProgressResults{completed, sent, arrived, total, tr, err}
		}

		// Add callback and channel to the sentTransferInfo
		sti[i].cbChan = cbChan
		sti[i].cb = cb

		// Add the transfer to the manager
		sti[i].tid, err = m.sent.AddTransfer(recipient, sti[i].key,
			sti[i].parts, sti[i].numFps, sti[i].cb, sti[i].period, sti[i].prng)
		if err != nil {
			t.Errorf("Failed to add sent transfer #%d: %+v", i, err)
		}

		// Add recipient as partner
		if addPartners {
			grp := m.store.E2e().GetGroup()
			dhKey := grp.NewInt(int64(i + 42))
			pubKey := diffieHellman.GeneratePublicKey(dhKey, grp)
			p := params.GetDefaultE2ESessionParams()
			rng := csprng.NewSystemRNG()
			_, mySidhPriv := util.GenerateSIDHKeyPair(
				sidh.KeyVariantSidhA, rng)
			theirSidhPub, _ := util.GenerateSIDHKeyPair(
				sidh.KeyVariantSidhB, rng)
			err = m.store.E2e().AddPartner(recipient, pubKey, dhKey,
				mySidhPriv, theirSidhPub, p, p)
			if err != nil {
				t.Errorf("Failed to add partner #%d %s: %+v", i, recipient, err)
			}
		}
	}

	// Add received transfers to manager and populate the receivedTransferInfo
	// list
	for i := range rti {
		// Generate PRNG, the file and its parts, and the transfer key
		prng := NewPrng(int64(42 + i))
		file, parts := newFile(numParts[i], partSize, prng, t)
		key, _ := ftCrypto.NewTransferKey(prng)

		// Create a receivedTransferInfo with all the transfer information
		rti[i] = receivedTransferInfo{
			key:      key,
			mac:      ftCrypto.CreateTransferMAC(file, key),
			parts:    parts,
			file:     file,
			fileSize: uint32(len(file)),
			numParts: numParts[i],
			numFps:   calcNumberOfFingerprints(numParts[i], 0.5),
			retry:    0.5,
			period:   time.Millisecond,
			prng:     prng,
		}

		// Create received progress callback and channel
		cbChan := make(chan receivedProgressResults, 8)
		cb := func(completed bool, received, total uint16,
			tr interfaces.FilePartTracker, err error) {
			cbChan <- receivedProgressResults{completed, received, total, tr, err}
		}

		// Add callback and channel to the receivedTransferInfo
		rti[i].cbChan = cbChan
		rti[i].cb = cb

		// Add the transfer to the manager
		rti[i].tid, err = m.received.AddTransfer(rti[i].key, rti[i].mac,
			rti[i].fileSize, rti[i].numParts, rti[i].numFps, rti[i].prng)
		if err != nil {
			t.Errorf("Failed to add received transfer #%d: %+v", i, err)
		}
	}

	return m, sti, rti
}

// receivedFtResults is used to return received new file transfer results on a
// channel from a callback.
type receivedFtResults struct {
	tid      ftCrypto.TransferID
	fileName string
	fileType string
	sender   *id.ID
	size     uint32
	preview  []byte
}

// sentProgressResults is used to return sent progress results on a channel from
// a callback.
type sentProgressResults struct {
	completed            bool
	sent, arrived, total uint16
	tracker              interfaces.FilePartTracker
	err                  error
}

// sentTransferInfo contains information on a sent transfer.
type sentTransferInfo struct {
	recipient *id.ID
	key       ftCrypto.TransferKey
	tid       ftCrypto.TransferID
	parts     [][]byte
	file      []byte
	numParts  uint16
	numFps    uint16
	retry     float32
	cb        interfaces.SentProgressCallback
	cbChan    chan sentProgressResults
	period    time.Duration
	prng      csprng.Source
}

// receivedProgressResults is used to return received progress results on a
// channel from a callback.
type receivedProgressResults struct {
	completed       bool
	received, total uint16
	tracker         interfaces.FilePartTracker
	err             error
}

// receivedTransferInfo contains information on a received transfer.
type receivedTransferInfo struct {
	key      ftCrypto.TransferKey
	tid      ftCrypto.TransferID
	mac      []byte
	parts    [][]byte
	file     []byte
	fileSize uint32
	numParts uint16
	numFps   uint16
	retry    float32
	cb       interfaces.ReceivedProgressCallback
	cbChan   chan receivedProgressResults
	period   time.Duration
	prng     csprng.Source
}

////////////////////////////////////////////////////////////////////////////////
// Test Network Manager                                                       //
////////////////////////////////////////////////////////////////////////////////

func newTestNetworkManager(sendErr bool, sendChan,
	sendE2eChan chan message.Receive, t *testing.T) interfaces.NetworkManager {
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
		rid:         0,
		messages:    make(map[id.Round][]message.TargetedCmixMessage),
		sendErr:     sendErr,
		health:      newTestHealthTracker(),
		sendChan:    sendChan,
		sendE2eChan: sendE2eChan,
	}
}

// testNetworkManager is a test implementation of NetworkManager interface.
type testNetworkManager struct {
	instance    *network.Instance
	updateRid   bool
	rid         id.Round
	messages    map[id.Round][]message.TargetedCmixMessage
	e2eMessages []message.Send
	sendErr     bool
	health      testHealthTracker
	sendChan    chan message.Receive
	sendE2eChan chan message.Receive
	sync.RWMutex
}

func (tnm *testNetworkManager) GetMsgList(rid id.Round) []message.TargetedCmixMessage {
	tnm.RLock()
	defer tnm.RUnlock()
	return tnm.messages[rid]
}

func (tnm *testNetworkManager) GetE2eMsg(i int) message.Send {
	tnm.RLock()
	defer tnm.RUnlock()
	return tnm.e2eMessages[i]
}

func (tnm *testNetworkManager) SendE2E(msg message.Send, _ params.E2E, _ *stoppable.Single) (
	[]id.Round, e2e.MessageID, time.Time, error) {
	tnm.Lock()
	defer tnm.Unlock()

	if tnm.sendErr {
		return nil, e2e.MessageID{}, time.Time{}, errors.New("SendE2E error")
	}

	tnm.e2eMessages = append(tnm.e2eMessages, msg)

	if tnm.sendE2eChan != nil {
		tnm.sendE2eChan <- message.Receive{
			Payload:     msg.Payload,
			MessageType: msg.MessageType,
			Sender:      &id.ID{},
			RecipientID: msg.Recipient,
		}
	}

	return []id.Round{0, 1, 2, 3}, e2e.MessageID{}, time.Time{}, nil
}

func (tnm *testNetworkManager) SendUnsafe(message.Send, params.Unsafe) ([]id.Round, error) {
	return []id.Round{}, nil
}

func (tnm *testNetworkManager) SendCMIX(format.Message, *id.ID, params.CMIX) (id.Round, ephemeral.Id, error) {
	return 0, ephemeral.Id{}, nil
}

func (tnm *testNetworkManager) SendManyCMIX(messages []message.TargetedCmixMessage, _ params.CMIX) (
	id.Round, []ephemeral.Id, error) {
	tnm.Lock()
	defer func() {
		// Increment the round every two calls to SendManyCMIX
		if tnm.updateRid {
			tnm.rid++
			tnm.updateRid = false
		} else {
			tnm.updateRid = true
		}
		tnm.Unlock()
	}()

	if tnm.sendErr {
		return 0, nil, errors.New("SendManyCMIX error")
	}

	tnm.messages[tnm.rid] = messages

	if tnm.sendChan != nil {
		for _, msg := range messages {
			tnm.sendChan <- message.Receive{
				Payload: msg.Message.Marshal(),
				Sender:  &id.ID{0},
				RoundId: tnm.rid,
			}
		}
	}

	return tnm.rid, nil, nil
}

type dummyEventMgr struct{}

func (d *dummyEventMgr) Report(int, string, string, string) {}
func (tnm *testNetworkManager) GetEventManager() interfaces.EventManager {
	return &dummyEventMgr{}
}

func (tnm *testNetworkManager) GetInstance() *network.Instance             { return tnm.instance }
func (tnm *testNetworkManager) GetHealthTracker() interfaces.HealthTracker { return tnm.health }
func (tnm *testNetworkManager) Follow(interfaces.ClientErrorReport) (stoppable.Stoppable, error) {
	return nil, nil
}
func (tnm *testNetworkManager) CheckGarbledMessages()        {}
func (tnm *testNetworkManager) InProgressRegistrations() int { return 0 }
func (tnm *testNetworkManager) GetSender() *gateway.Sender   { return nil }
func (tnm *testNetworkManager) GetAddressSize() uint8        { return 0 }
func (tnm *testNetworkManager) RegisterAddressSizeNotification(string) (chan uint8, error) {
	return nil, nil
}
func (tnm *testNetworkManager) UnregisterAddressSizeNotification(string) {}
func (tnm *testNetworkManager) SetPoolFilter(gateway.Filter)             {}
func (tnm *testNetworkManager) GetVerboseRounds() string                 { return "" }

type testHealthTracker struct {
	chIndex, fnIndex uint64
	channels         map[uint64]chan bool
	funcs            map[uint64]func(bool)
	healthy          bool
}

////////////////////////////////////////////////////////////////////////////////
// Test Health Tracker                                                        //
////////////////////////////////////////////////////////////////////////////////

func newTestHealthTracker() testHealthTracker {
	return testHealthTracker{
		chIndex:  0,
		fnIndex:  0,
		channels: make(map[uint64]chan bool),
		funcs:    make(map[uint64]func(bool)),
		healthy:  true,
	}
}

func (tht testHealthTracker) AddChannel(c chan bool) uint64 {
	tht.channels[tht.chIndex] = c
	tht.chIndex++
	return tht.chIndex - 1
}

func (tht testHealthTracker) RemoveChannel(chanID uint64) { delete(tht.channels, chanID) }

func (tht testHealthTracker) AddFunc(f func(bool)) uint64 {
	tht.funcs[tht.fnIndex] = f
	tht.fnIndex++
	return tht.fnIndex - 1
}

func (tht testHealthTracker) RemoveFunc(funcID uint64) { delete(tht.funcs, funcID) }
func (tht testHealthTracker) IsHealthy() bool          { return tht.healthy }
func (tht testHealthTracker) WasHealthy() bool         { return tht.healthy }

////////////////////////////////////////////////////////////////////////////////
// NDF Primes                                                                 //
////////////////////////////////////////////////////////////////////////////////

func getNDF() *ndf.NetworkDefinition {
	return &ndf.NetworkDefinition{
		E2E: ndf.Group{
			Prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A" +
				"8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3D" +
				"D2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E78615" +
				"75E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC" +
				"6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C" +
				"4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F2" +
				"6E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE" +
				"448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E" +
				"198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FF" +
				"DDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323" +
				"631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C" +
				"3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E63" +
				"19BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC3" +
				"5873847AEF49F66E43873",
			Generator: "2",
		},
		CMIX: ndf.Group{
			Prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642" +
				"F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757" +
				"264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F" +
				"9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091E" +
				"B51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D" +
				"0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D3" +
				"92145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A" +
				"2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7" +
				"995FAD5AABBCFBE3EDA2741E375404AE25B",
			Generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E2480" +
				"9670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D" +
				"1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A33" +
				"8661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361" +
				"C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B28" +
				"5DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD929" +
				"59859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D83" +
				"2186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8" +
				"B6F116F7AD9CF505DF0F998E34AB27514B0FFE7",
		},
	}
}
