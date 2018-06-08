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

// Returns the entire value of the wallet
func (w *Wallet) Value() int {

	value := uint32(0)

	w.lock.Lock()

	//Sums the value of all coins
	for indx, d := range *w.storage {
		value += (uint32(1) << uint32(indx)) * uint32(len(d))
	}

	w.lock.Unlock()

	return int(value)
}

// Withdraws coins equal to the value passed
func (w *Wallet) withdraw(value uint32) ([]Coin, error) {

	w.lock.Lock()

	//Copy the old state of the wallet
	storageCopy := w.copyStorage()

	if value > uint32(w.Value()) {
		return nil, ErrCannotFund
	}

	var coinList []Coin

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 && len((*w.storage)[i]) < 1 {
			return nil, ErrIncorrectChange
		}
	}

	for i := uint32(0); i < uint32(8); i++ {
		d := value >> i

		if d == 1 {
			coinList = append(coinList, (*w.storage)[i][0])
			(*w.storage)[i] = (*w.storage)[i][1:len((*w.storage)[i])]
		}
	}

	err := globals.Session.StoreSession()

	if err != nil {
		*w.storage = storageCopy
		return nil, err
	}

	w.lock.Unlock()

	return coinList, nil
}

// deposits the coins passed if valid
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
		(*w.storage)[c.Denomination] = append((*w.storage)[c.Denomination], *c)
	}

	err = globals.Session.StoreSession()

	if err != nil {
		*w.storage = storageCopy
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
