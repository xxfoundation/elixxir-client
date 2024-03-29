////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"io"
	"math/rand"
	"testing"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// PRNG                                                                       //
////////////////////////////////////////////////////////////////////////////////

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }

////////////////////////////////////////////////////////////////////////////////
// Test Managers                                                              //
////////////////////////////////////////////////////////////////////////////////

// newTestManager creates a new Manager that has groups stored for testing. One
// of the groups in the list is also returned.
func newTestManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	t *testing.T) *Manager {
	store := storage.InitTestingSession(t)
	payloadSize := store.GetCmixGroup().GetP().ByteLen()
	n := uint64(0)
	m := &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		statusChan:     make(chan bool, statusChanLen),
		store:          store,
		net:            newMockCmix(payloadSize),
		rng:            fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
		totalSent:      &n,
	}

	return m
}

// generateMessage is a utility function which generates a format.Message
// given message data.
func generateMessage(payloadSize int,
	fingerprint format.Fingerprint,
	service cmix.Service,
	payload, mac []byte) format.Message {

	// Build message. Will panic if inputs are not correct.
	msg := format.NewMessage(payloadSize)
	msg.SetContents(payload)
	msg.SetKeyFP(fingerprint)
	SIH, err := service.Hash(nil, msg.GetContents())
	if err != nil {
		panic(err)
	}
	msg.SetSIH(SIH)
	msg.SetMac(mac)

	return msg
}
