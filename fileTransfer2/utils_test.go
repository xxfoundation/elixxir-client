////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"bytes"
	"encoding/binary"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"io"
	"math/rand"
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

////////////////////////////////////////////////////////////////////////////////
// Mock cMix Cmix                                                           //
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
	round         id.Round
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
		round:         0,
		healthIndex:   0,
	}
}

func (m *mockCmix) GetMaxMessageLength() int {
	msg := format.NewMessage(m.numPrimeBytes)
	return msg.ContentsSize()
}

func (m *mockCmix) SendMany(messages []cmix.TargetedCmixMessage,
	_ cmix.CMIXParams) (id.Round, []ephemeral.Id, error) {
	m.handler.Lock()
	defer m.handler.Unlock()
	round := m.round
	m.round++
	for _, targetedMsg := range messages {
		msg := format.NewMessage(m.numPrimeBytes)
		msg.SetContents(targetedMsg.Payload)
		msg.SetMac(targetedMsg.Mac)
		msg.SetKeyFP(targetedMsg.Fingerprint)
		m.handler.processorMap[targetedMsg.Fingerprint].Process(msg,
			receptionID.EphemeralIdentity{Source: targetedMsg.Recipient},
			rounds.Round{ID: round})
	}
	return round, []ephemeral.Id{}, nil
}

func (m *mockCmix) AddFingerprint(_ *id.ID, fp format.Fingerprint, mp message.Processor) error {
	m.handler.Lock()
	defer m.handler.Unlock()
	m.handler.processorMap[fp] = mp
	return nil
}

func (m *mockCmix) DeleteFingerprint(_ *id.ID, fp format.Fingerprint) {
	m.handler.Lock()
	defer m.handler.Unlock()
	delete(m.handler.processorMap, fp)
}

func (m *mockCmix) CheckInProgressMessages() {}

func (m *mockCmix) IsHealthy() bool {
	return m.health
}

func (m *mockCmix) WasHealthy() bool { return true }

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

func (m *mockCmix) GetRoundResults(_ time.Duration,
	roundCallback cmix.RoundEventCallback, rids ...id.Round) error {
	go roundCallback(true, false, map[id.Round]cmix.RoundResult{rids[0]: {}})
	return nil
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

func (m *mockStorage) GetKV() *versioned.KV        { return m.kv }
func (m *mockStorage) GetCmixGroup() *cyclic.Group { return m.cmixGroup }
