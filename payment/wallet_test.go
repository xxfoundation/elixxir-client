package payment

import (
	"gitlab.com/privategrity/crypto/coin"
	"math"
	"testing"
)

func TestWalletStorage(t *testing.T) {
	wallet := NewWalletStorage()

	values := []uint8{2, 6, 3, 2, 1, 4, 5, 7, 7, 4, 2, 4, 2}
	expectedValue := uint32(0)

	for _, denomination := range values {
		c, _ := NewCoin(denomination)
		wallet.Deposit([]*Coin{c})
		expectedValue += 1 << denomination
		if wallet.Value() != expectedValue {
			t.Errorf("WalletStorage.Value(): Value did not match,"+
				" Expected: %v, Recieved: %v", expectedValue, wallet.Value())
		}
	}

	cLst, err := wallet.Withdraw(2)

	if err != nil {
		t.Errorf("WalletStorage.Withdraw(): Did not return withdral,"+
			" on valid withdrawl: %s", err.Error())
	} else {
		if len(cLst) != 1 {
			t.Errorf("WalletStorage.Withdraw(): Did not withdraw correct"+
				"coins; len: %v", len(cLst))
		}
		if !cLst[0].Validate() {
			t.Errorf("WalletStorage.Withdraw(): Did not withdraw valid" +
				"coins")
		}

		for _, c := range wallet.Coins[2] {
			if compareCoins(&c, &cLst[0]) {
				t.Errorf("WalletStorage.Withdraw(" +
					"): Coin not removed from wallet")
			}
		}
	}

	_, err = wallet.Withdraw(math.MaxUint32)

	if err == nil {
		t.Errorf("WalletStorage.Withdraw(" +
			"): Allowed withdrawl on invalid ammount")
	}

	wallet = NewWalletStorage()

	c0, _ := NewCoin(0)
	c1, _ := NewCoin(0)

	wallet.Deposit([]*Coin{c0, c1})

	_, err = wallet.Withdraw(2)

	if err == nil {
		t.Errorf("WalletStorage.Withdraw(" +
			"): Allowed withdrawl with incorrect change")
	}
}

func compareCoins(a, b *Coin) bool {
	if a.Denomination != a.Denomination {
		return false
	}

	for i := 0; i < coin.CoinLen; i++ {
		if a.Image[i] != b.Image[i] {
			return false
		}
		if a.Preimage[i] != b.Preimage[i] {
			return false
		}
	}

	return true
}
