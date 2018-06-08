package payment

import (
	"encoding/gob"
	"errors"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/coin"
	"sync"
)

const NumDenominations = 8
const WalletStorageKey = "WalletStorage"

var ErrCannotFund = errors.New("not enough coins in wallet to fund")
var ErrIncorrectChange = errors.New("coins in incorrect denominations to fund")
var ErrInvalidCoin = errors.New("coin is not valid")

// Stores the actual coins
type WalletStorage [NumDenominations][]Coin

// Struct which wil exposed
type Wallet struct {
	storage **WalletStorage
	lock    *sync.Mutex
}

// Returns a wallet.  it is loaded from the session object if one exists,
// otherwise a new one is created and stored in the session object
func NewWallet() (*Wallet, error) {
	var w WalletStorage

	ws := &w

	gob.Register(WalletStorage{})

	// gets the wallet from the session
	wsi, err := globals.Session.QueryMap(WalletStorageKey)
	if err != nil {
		//make a new session object if none exists
		if err == globals.ErrQuery {

			for i := 0; i < int(coin.Denominations); i++ {
				(*ws)[i] = make([]Coin, 0)
			}

			err = globals.Session.UpsertMap(WalletStorageKey, &ws)
		}
		if err != nil {
			return nil, err
		}
	} else {
		ws = *(wsi.(**WalletStorage))
	}

	return &Wallet{&ws, &sync.Mutex{}}, nil
}

// Returns the entire value of the wallet without a lock
func (w *Wallet) Value() int {

	w.lock.Lock()
	v := w.value()
	w.lock.Unlock()

	return int(v)
}

// Returns the entire value of the wallet with a lock
func (w *Wallet) value() uint32 {

	value := uint32(0)

	//Sums the value of all coins
	for indx, d := range *w.storage {
		value += (uint32(1) << uint32(indx)) * uint32(len(d))
	}

	return value
}

// Withdraws coins equal to the value passed
func (w *Wallet) withdraw(value uint32) ([]Coin, error) {

	w.lock.Lock()

	//Copy the old state of the wallet
	storageCopy := w.copyStorage()

	if value > w.value() {
		return nil, ErrCannotFund
	}

	var coinList []Coin

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 && len((storageCopy)[i]) < 1 {
			return nil, ErrIncorrectChange
		}
	}

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 {
			coinList = append(coinList, (storageCopy)[i][0])
			(storageCopy)[i] = (*w.storage)[i][1:len((storageCopy)[i])]
		}
	}

	oldStorage := *w.storage

	*w.storage = storageCopy

	err := globals.Session.StoreSession()

	if err != nil {
		*w.storage = oldStorage
		return nil, err
	}

	w.lock.Unlock()

	return coinList, nil
}

// adds the coins passed if all are valid
func (w *Wallet) deposit(coins []*Coin) error {

	var err error

	w.lock.Lock()

	storageCopy := w.copyStorage()

	for _, c := range coins {
		if !c.Validate() {
			return ErrInvalidCoin
		}
	}

	for _, c := range coins {
		(storageCopy)[c.Denomination] = append((storageCopy)[c.Denomination], *c)
	}

	oldStorage := *w.storage

	*w.storage = storageCopy

	err = globals.Session.StoreSession()

	if err != nil {
		*w.storage = oldStorage
	}

	w.lock.Unlock()

	return err

}

// copies the wallet storage object
func (w *Wallet) copyStorage() *WalletStorage {
	var ws WalletStorage

	for i := 0; i < int(coin.Denominations); i++ {
		ws[i] = make([]Coin, len((**w.storage)[i]))
		copy(ws[i], (*w.storage)[i])
	}

	return &ws
}
