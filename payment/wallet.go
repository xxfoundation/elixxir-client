package payment

import (
	"errors"
)

const NumDenominations = 8

var ErrCannotFund = errors.New("not enough coins in wallet to fund")
var ErrIncorrectChange = errors.New("coins in incorrect denominations to fund")
var ErrInvalidCoin = errors.New("coin is not valid")

type WalletStorage struct {
	Coins [NumDenominations][]Coin
}

func NewWalletStorage() *WalletStorage {
	var a [8][]Coin
	return &WalletStorage{a}
}

func (w *WalletStorage) Value() uint32 {

	value := uint32(0)
	for indx, d := range w.Coins {
		value += (uint32(1) << uint32(indx)) * uint32(len(d))
	}

	return value
}

func (w *WalletStorage) Withdraw(value uint32) ([]Coin, error) {
	if value > w.Value() {
		return nil, ErrCannotFund
	}

	var coinList []Coin

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 && len(w.Coins[i]) < 1 {
			return nil, ErrIncorrectChange
		}
	}

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 {
			coinList = append(coinList, w.Coins[i][0])
			w.Coins[i] = w.Coins[i][1:len(w.Coins[i])]
		}
	}

	return coinList, nil
}

func (w *WalletStorage) Deposit(coins []*Coin) error {
	for _, c := range coins {
		if !c.Validate() {
			return ErrInvalidCoin
		}
	}

	for _, c := range coins {
		w.Coins[c.Denomination] = append(w.Coins[c.Denomination], *c)
	}

	return nil

}
