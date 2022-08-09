////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/crypto/csprng"
	"io"
	"math/rand"
	"testing"
	"time"
)

// ////////////////////////////////////////////////////////////////////////////////
// // PRNG                                                                       //
// ////////////////////////////////////////////////////////////////////////////////

// Prng is a PRNG that satisfies the csprng.Source interface.
type Prng struct{ prng io.Reader }

func NewPrng(seed int64) csprng.Source     { return &Prng{rand.New(rand.NewSource(seed))} }
func (s *Prng) Read(b []byte) (int, error) { return s.prng.Read(b) }
func (s *Prng) SetSeed([]byte) error       { return nil }

// ////////////////////////////////////////////////////////////////////////////////
// // Test Managers                                                              //
// ////////////////////////////////////////////////////////////////////////////////

// newTestManager creates a new Manager that has groups stored for testing. One
// of the groups in the list is also returned.
func newTestManager(maxNumMessages int, avgSendDelta, randomRange time.Duration,
	sendErr bool, t *testing.T) *Manager {
	store := storage.InitTestingSession(t)
	m := &Manager{
		maxNumMessages: maxNumMessages,
		avgSendDelta:   avgSendDelta,
		randomRange:    randomRange,
		statusChan:     make(chan bool, statusChanLen),
		store:          &store,
		rng:            fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG),
	}

	return m
}
