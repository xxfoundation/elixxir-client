package payment

import (
	"gitlab.com/privategrity/crypto/coin"
	"errors"
	"sync"
	"encoding/gob"
	"gitlab.com/privategrity/client/user"
)

type OrderedCoinStorage struct{
	list *[]coin.Sleeve
	mutex sync.Mutex
	value uint64
}

var ErrInsufficientFunds = errors.New("not enough funds to fund request")
var ErrInvalidOrganizationOfFunds = errors.New("cannot fit requested funds within MaxFunds")

var NilSleeve = coin.Sleeve{}

func NewOrderedStorage(tag string)(*OrderedCoinStorage, error){
	gob.Register(OrderedCoinStorage{})

	var osclPtr *[]coin.Sleeve

	oscli, err := user.TheSession.QueryMap(tag)
	if err!=nil{
		//If there is an err make the object
		osl := make([]coin.Sleeve,0)
		osclPtr = &osl

		if err == user.ErrQuery {
			err = user.TheSession.UpsertMap(tag, &osclPtr)
		}
		if err != nil {
			return nil, err
		}
	}else{
		osclPtr = oscli.(*[]coin.Sleeve)
	}

	value := uint64(0)

	for _,cs := range *osclPtr {
		value += cs.Value()
	}

	return &OrderedCoinStorage{list: osclPtr, value:value}, nil
}


func (ocs *OrderedCoinStorage) Value()uint64{
	ocs.mutex.Lock()
	v := ocs.value
	ocs.mutex.Unlock()
	return v
}

func (ocs *OrderedCoinStorage) add(cs coin.Sleeve){
	if len(*ocs.list) == 0{
		*ocs.list = append(*ocs.list,cs)
	}else{
		for i:=0;i<len(*ocs.list);i++{
			if (*ocs.list)[i].Value()>cs.Value() {
				tmp := append((*ocs.list)[:i], cs)
				*ocs.list = append(tmp, (*ocs.list)[i:]...)
			}
		}
	}

	ocs.value += cs.Value()
}

func (ocs *OrderedCoinStorage) Add(cs coin.Sleeve){
	ocs.mutex.Lock()
	ocs.add(cs)
	ocs.mutex.Unlock()
}

func (ocs *OrderedCoinStorage) pop(index uint64)coin.Sleeve{
	if uint64(len(*ocs.list))>=index{
		return coin.Sleeve{}
	}

	cs := (*ocs.list)[index]

	*ocs.list = append((*ocs.list)[:index],(*ocs.list)[index+1:]...)

	ocs.value -= cs.Value()

	return cs
}

func (ocs *OrderedCoinStorage) Pop(index uint64)coin.Sleeve{
	ocs.mutex.Lock()
	cs := ocs.Pop(index)
	ocs.mutex.Unlock()
	return cs
}

func (ocs *OrderedCoinStorage) get(index uint64)coin.Sleeve{
	if uint64(len(*ocs.list))>=index{
		return coin.Sleeve{}
	}

	return (*ocs.list)[index]
}

func (ocs *OrderedCoinStorage) Get(index uint64)coin.Sleeve{
	ocs.mutex.Lock()
	cs := ocs.get(index)
	ocs.mutex.Unlock()
	return cs
}

func (ocs *OrderedCoinStorage) Fund(value, maxCoins uint64)([]coin.Sleeve, coin.Sleeve, error){
	ocs.mutex.Lock()

	// Return an error if there are insufficient funds
	if value> ocs.value{
		ocs.mutex.Unlock()
		return []coin.Sleeve{}, NilSleeve, ErrInsufficientFunds
	}

	// Reduce max coins if it is greater than the total number of coins
	if maxCoins > uint64(len(*ocs.list)){
		maxCoins = uint64(len(*ocs.list))
	}

	// Create variables
	var funds []coin.Sleeve
	sum := uint64(0)

	// Step 1: Fill with all smallest coins
	for i:=uint64(0);i< maxCoins;i++{
		cs := ocs.pop(0)
		funds = append(funds, cs)
		sum += cs.Value()
		if sum>=value{
			goto Success
		}
	}

	// Step 2: unwind and remove each coin from the highest to
	// lowest
	for i:= maxCoins -1;i>=0;i--{
		j:=uint64(0)
		newSum := uint64(0)
		for j<uint64(len(funds)){
			newSum = sum-funds[i].Value()+ ocs.get(j).Value()
			if newSum>=value{
				break
			}
			j++
		}
		oldSleeve := funds[i]
		funds[i] = ocs.pop(j)
		ocs.add(oldSleeve)
		sum = newSum
		if sum>=value{
			goto Success
		}
	}

	// Step 3: If nothing is found, add funds back onto the ordered list,
	// it will be all the highest coins so it can just be appended
	*ocs.list = append(*ocs.list,funds...)
	ocs.value += sum
	ocs.mutex.Unlock()

	return []coin.Sleeve{},NilSleeve,ErrInvalidOrganizationOfFunds

	Success:

		change := NilSleeve

		if sum>value{
			var err error
			change, err = coin.NewSleeve(sum-value)
			if err!=nil{
				for _,c := range funds{
					ocs.add(c)
				}
				return []coin.Sleeve{},NilSleeve,err
			}
		}

		ocs.mutex.Unlock()
		return funds, change, nil
}