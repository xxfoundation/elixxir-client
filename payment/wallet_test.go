////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"math"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	globals.InitStorage(&globals.RamStorage{}, "")
	user.TheSession = user.NewSession(nil, "abc", nil)
	os.Exit(m.Run())
}

// Guarantees it is impossible to withdraw from an empty wallet
func TestWallet_withdraw_empty(t *testing.T) {
	wallet, _ := NewWallet()

	// Make sure the wallet is empty
	emptyWallet := &WalletStorage{}
	wallet.storage = &(emptyWallet)

	// Attempt to withdraw from the empty wallet
	_, err := wallet.withdraw(2)

	if err != ErrCannotFund {
		t.Errorf("Wallet.withdraw: Expected an error withdrawing from empty" +
			" wallet.")
	}
}

// Guarantees it is impossible to withdraw an invalid denomination
func TestWallet_withdraw_invalid(t *testing.T) {
	wallet, _ := NewWallet()

	// Create large denomination so smaller denomination can't be withdrawn
	testWallet := &WalletStorage{}
	testCoin, _ := NewCoin(2)
	testCoin2, _ := NewCoin(2)
	testWallet[2] = append(testWallet[2], *testCoin)
	testWallet[2] = append(testWallet[2], *testCoin2)
	wallet.storage = &(testWallet)

	// Attempt to withdraw a small denomination
	_, err := wallet.withdraw(2)

	if err != ErrIncorrectChange {
		t.Errorf("Wallet.withdraw: Expected an error withdrawing invalid" +
			" amount from wallet.")
	}
}

func TestWallet(t *testing.T) {
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
		t.Errorf("Wallet.withdraw(): Did not return withdrawal"+
			" on valid withdrawal: %s", err.Error())
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
			"): Allowed withdrawal on invalid ammount")
	}

	user.TheSession.DeleteMap(WalletStorageKey)

	wallet, err = NewWallet()

	c0, _ := NewCoin(0)
	c1, _ := NewCoin(0)

	wallet.deposit([]*Coin{c0, c1})

	_, err = wallet.withdraw(2)

	if err == nil {
		t.Errorf("Wallet.withdraw(" +
			"): Allowed withdrawal with incorrect change")
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
