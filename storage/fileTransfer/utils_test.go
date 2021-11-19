////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/crypto/csprng"
	"io"
	"math/rand"
)

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
