package payment

import (
	"gitlab.com/privategrity/crypto/coin"
	"math"
	"testing"
)

func TestCoin(t *testing.T) {
	//Test Valid Denominations
	for i := uint8(0); i < coin.Denominations; i++ {
		c, err := NewCoin(i)
		if err != nil {
			t.Errorf("NewCoin(): Valid coin with denomination of %v could"+
				" not be created: %s", i, err.Error())
		}
		if c.Preimage == nil {
			t.Errorf("NewCoin(): Valid coin with denomination of %v did"+
				" not return preimage", i)
		}
		if c.Image == nil {
			t.Errorf("NewCoin(): Valid coin with denomination of %v did"+
				" not return image", i)
		}
		if !c.Validate() {
			t.Errorf("NewCoin() and Coin.Validate()"+
				": Valid coin with denomination of %v did"+
				" not return as valid", i)
		}

	}

	//Test Invalid Denominations
	for i := coin.Denominations; i < math.MaxUint8; i++ {
		c, err := NewCoin(i)
		if err == nil {
			t.Errorf("NewCoin(): Invalid coin with denomination of %v"+
				"returned as valid", i)
		}
		if c != nil {
			t.Errorf("NewCoin(): Invalid coin with denomination of %v"+
				"returned with; preimage: %v, image: %v ", i, c.Preimage, c.Image)
		}
	}

	//Test Validate on invalid coin
	c, _ := NewCoin(1)
	c.Image[0] = c.Image[0] + 1

	if c.Validate() {
		t.Errorf("Coin.Validate(): Validated Invalid coin")
	}
}
