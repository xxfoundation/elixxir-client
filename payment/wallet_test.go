package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/coin"
	"math"
	"testing"
)

func TestWallet(t *testing.T) {

	globals.InitStorage(&globals.RamStorage{}, "")

	globals.Session = globals.NewUserSession(nil, "abc", nil)

	wallet, err := NewWallet()

	if err != nil {
		t.Errorf("NewWallet(): Could not make new wallet: %s", err.Error())
	}

	values := []uint8{2, 6, 3, 2, 1, 4, 5, 7, 7, 4, 2, 4, 2}
	expectedValue := uint32(0)

	for _, denomination := range values {
		c, _ := NewCoin(denomination)
		wallet.deposit([]*Coin{c})
		expectedValue += 1 << denomination
		if wallet.Value() != int(expectedValue) {
			t.Errorf("Wallet.Value(): Value did not match,"+
				" Expected: %v, Recieved: %v", expectedValue, wallet.Value())
		}
	}

	cLst, err := wallet.withdraw(2)

	if err != nil {
		t.Errorf("Wallet.ithdraw(): Did not return withdral,"+
			" on valid withdrawl: %s", err.Error())
	} else {
		if len(cLst) != 1 {
			t.Errorf("Wallet.withdraw(): Did not withdraw correct"+
				"coins; len: %v", len(cLst))
		}
		if !cLst[0].Validate() {
			t.Errorf("Wallet.withdraw(): Did not withdraw valid" +
				"coins")
		}

		for _, c := range (*wallet.storage)[2] {
			if compareCoins(&c, &cLst[0]) {
				t.Errorf("Wallet.withdraw(" +
					"): Coin not removed from wallet")
			}
		}
	}

	_, err = wallet.withdraw(math.MaxUint32)

	if err == nil {
		t.Errorf("Wallet.withdraw(" +
			"): Allowed withdrawl on invalid ammount")
	}

	globals.Session.DeleteMap(WalletStorageKey)

	wallet, err = NewWallet()

	c0, _ := NewCoin(0)
	c1, _ := NewCoin(0)

	wallet.deposit([]*Coin{c0, c1})

	_, err = wallet.withdraw(2)

	if err == nil {
		t.Errorf("Wallet.withdraw(" +
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
