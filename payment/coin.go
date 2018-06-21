////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"errors"
	"gitlab.com/privategrity/crypto/coin"
)

var ErrInvalidDenomination = errors.New("invalid denomination passed")

type Coin struct {
	Preimage     *coin.Preimage
	Image        *coin.Image
	Denomination uint8
}

func NewCoin(denomination uint8) (*Coin, error) {

	if denomination >= coin.Denominations {
		return nil, ErrInvalidDenomination
	}

	c := Coin{}
	pi, err := coin.NewCoinPreimage(denomination)

	c.Preimage = &pi

	if err != nil {
		return nil, err
	}

	img := ((*c.Preimage).ComputeImage())

	c.Image = &img

	c.Denomination = denomination

	return &c, nil
}

func (c *Coin) Validate() bool {
	if c.Denomination >= coin.Denominations {
		return false
	}

	if c.Preimage != nil && !c.Image.Verify(*c.Preimage) {
		return false
	}

	return true
}
