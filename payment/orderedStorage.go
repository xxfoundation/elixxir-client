////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"errors"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
)

type OrderedCoinStorage struct {
	list  *[]coin.Sleeve
	value uint64

	session user.Session
}

var ErrInsufficientFunds = errors.New("not enough funds to fund request")
var ErrInvalidOrganizationOfFunds = errors.New("cannot fit requested funds within MaxFunds")

var NilSleeve = coin.Sleeve{}

// Checks to see if an ordered storage of the given tag is present in session.  If one is, then it returns it.
// If one isn't, then a new one is created
func CreateOrderedStorage(tag string, session user.Session) (*OrderedCoinStorage, error) {
	var osclPtr *[]coin.Sleeve

	oscli, err := session.QueryMap(tag)
	if err != nil {
		//If there is an err make the object
		osl := make([]coin.Sleeve, 0)
		osclPtr = &osl

		if err == user.ErrQuery {
			err = session.UpsertMap(tag, osclPtr)
		}
		if err != nil {
			return nil, err
		}
	} else {
		osclPtr = oscli.(*[]coin.Sleeve)
	}

	value := uint64(0)

	for _, cs := range *osclPtr {
		value += cs.Value()
	}

	return &OrderedCoinStorage{list: osclPtr, value: value, session: session}, nil
}

// Returns the value of all coins in the ordered storage
func (ocs *OrderedCoinStorage) Value() uint64 {
	ocs.session.LockStorage()
	v := ocs.value
	ocs.session.UnlockStorage()
	return v
}

// Adds a coin to the ordered storage
func (ocs *OrderedCoinStorage) Add(cs coin.Sleeve) {
	ocs.session.LockStorage()
	ocs.add(cs)
	ocs.session.UnlockStorage()
}

// gets the coin at a specific index in the ordered storage
func (ocs *OrderedCoinStorage) Get(index uint64) (coin.Sleeve, bool) {
	ocs.session.LockStorage()
	cs, b := ocs.get(index)
	ocs.session.UnlockStorage()
	return cs, b
}

// pops a coin at the specific index int eh ordered storage
func (ocs *OrderedCoinStorage) Pop(index uint64) (coin.Sleeve, bool) {
	ocs.session.LockStorage()
	cs, b := ocs.pop(index)
	ocs.session.UnlockStorage()
	return cs, b
}

// Funds coins up to the requested amount with change which stores the excess
func (ocs *OrderedCoinStorage) Fund(value, maxCoins uint64) ([]coin.Sleeve, coin.Sleeve, error) {
	ocs.session.LockStorage()

	// Return an error if there are insufficient funds
	if value > ocs.value {
		ocs.session.UnlockStorage()
		return []coin.Sleeve{}, NilSleeve, ErrInsufficientFunds
	}

	// Reduce max coins if it is greater than the total number of coins
	if maxCoins > uint64(len(*ocs.list)) {
		maxCoins = uint64(len(*ocs.list))
	}

	// Create variables
	var funds []coin.Sleeve
	sum := uint64(0)

	// Step 1: Fill with all smallest coins
	for i := uint64(0); i < maxCoins; i++ {
		cs, _ := ocs.pop(0)
		funds = append(funds, cs)
		sum += cs.Value()
		if sum >= value {
			goto Success
		}
	}

	// Step 2: unwind and remove each coin from the highest to
	// lowest
	for i := int64(maxCoins) - 1; i >= 0; i-- {
		j := int64(-1)
		newSum := uint64(0)
		for j < int64(len(*ocs.list)-1) {
			j++
			csg, _ := ocs.get(uint64(j))
			newSum = sum - funds[i].Value() + csg.Value()
			if newSum >= value {
				break
			}
		}

		oldSleeve := funds[i]
		funds[i], _ = ocs.pop(uint64(j))
		ocs.add(oldSleeve)
		sum = newSum
		if sum >= value {
			goto Success
		}
	}

	// Step 3: If nothing is found, add funds back onto the ordered list,
	// it will be all the highest coins so it can just be appended
	*ocs.list = append(*ocs.list, funds...)
	ocs.value += sum
	ocs.session.UnlockStorage()

	return []coin.Sleeve{}, NilSleeve, ErrInvalidOrganizationOfFunds

Success:

	change := NilSleeve

	if sum > value {
		var err error
		change, err = coin.NewSleeve(sum - value)
		if err != nil {
			for _, c := range funds {
				ocs.add(c)
			}
			return []coin.Sleeve{}, NilSleeve, err
		}
	}

	ocs.session.UnlockStorage()
	return funds, change, nil
}

// INTERNAL FUNCTIONS
func (ocs *OrderedCoinStorage) add(cs coin.Sleeve) {

	i := 0

	for i < len(*ocs.list) {
		if cs.Value() < (*ocs.list)[i].Value() {
			break
		}
		i++
	}

	newList := make([]coin.Sleeve, len(*ocs.list)+1)

	copy(newList[:i], (*ocs.list)[:i])
	newList[i] = cs
	copy(newList[i+1:], (*ocs.list)[i:])

	*ocs.list = newList

	ocs.value += cs.Value()
}

func (ocs *OrderedCoinStorage) get(index uint64) (coin.Sleeve, bool) {
	if index >= uint64(len(*ocs.list)) {
		return coin.Sleeve{}, false
	}

	return (*ocs.list)[index], true
}

func (ocs *OrderedCoinStorage) pop(index uint64) (coin.Sleeve, bool) {
	cs, b := ocs.get(index)

	if b {
		*ocs.list = append((*ocs.list)[:index], (*ocs.list)[index+1:]...)

		ocs.value -= cs.Value()
	}

	return cs, b
}
