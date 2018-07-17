package payment

import (
	"gitlab.com/privategrity/crypto/coin"
	"errors"
	"sync"
	"encoding/gob"
	"gitlab.com/privategrity/client/user"
)

type OrderedStorage struct{
	list *[]coin.Sleeve
	mutex sync.Mutex
	value uint64
}

var ErrInsufficientFunds = errors.New("not enough funds to fund request")
var ErrInvalidOrganizationOfFunds = errors.New("cannot fit requested funds within MaxFunds")

var NilSleeve = coin.Sleeve{}

func NewOrderedStorage(tag string)(*OrderedStorage, error){
	gob.Register(OrderedStorage{})

	var oslPtr *[]coin.Sleeve

	osli, err := user.TheSession.QueryMap(tag)
	if err!=nil{
		//If there is an err make the object
		osl := make([]coin.Sleeve,0)
		oslPtr = &osl

		if err == user.ErrQuery {
			err = user.TheSession.UpsertMap(tag, &oslPtr)
		}
		if err != nil {
			return nil, err
		}
	}else{
		oslPtr = osli.(*[]coin.Sleeve)
	}

	value := uint64(0)

	for _,cs := range *oslPtr{
		value += cs.Value()
	}

	return &OrderedStorage{list: oslPtr, value:value}, nil
}


func (os *OrderedStorage) Value()uint64{
	os.mutex.Lock()
	v := os.value
	os.mutex.Unlock()
	return v
}

func (os *OrderedStorage) add(cs coin.Sleeve){
	if len(*os.list) == 0{
		*os.list = append(*os.list,cs)
	}else{
		for i:=0;i<len(*os.list);i++{
			if (*os.list)[i].Value()>cs.Value() {
				tmp := append((*os.list)[:i], cs)
				*os.list = append(tmp, (*os.list)[i:]...)
			}
		}
	}

	os.value += cs.Value()
}

func (os *OrderedStorage) Add(cs coin.Sleeve){
	os.mutex.Lock()
	os.add(cs)
	os.mutex.Unlock()
}

func (os *OrderedStorage) pop(index uint64)coin.Sleeve{
	if uint64(len(*os.list))>=index{
		return coin.Sleeve{}
	}

	cs := (*os.list)[index]

	*os.list = append((*os.list)[:index],(*os.list)[index+1:]...)

	os.value -= cs.Value()

	return cs
}

func (os *OrderedStorage) Pop(index uint64)coin.Sleeve{
	os.mutex.Lock()
	cs := os.Pop(index)
	os.mutex.Unlock()
	return cs
}

func (os *OrderedStorage) get(index uint64)coin.Sleeve{
	if uint64(len(*os.list))>=index{
		return coin.Sleeve{}
	}

	return (*os.list)[index]
}

func (os *OrderedStorage) Get(index uint64)coin.Sleeve{
	os.mutex.Lock()
	cs := os.get(index)
	os.mutex.Unlock()
	return cs
}

func (os *OrderedStorage) Fund(value, maxCoins uint64)([]coin.Sleeve, coin.Sleeve, error){
	os.mutex.Lock()

	// Return an error if there are insufficient funds
	if value>os.value{
		os.mutex.Unlock()
		return []coin.Sleeve{}, NilSleeve, ErrInsufficientFunds
	}

	// Reduce max coins if it is greater than the total number of coins
	if maxCoins > uint64(len(*os.list)){
		maxCoins = uint64(len(*os.list))
	}

	// Create variables
	var funds []coin.Sleeve
	sum := uint64(0)

	// Step 1: Fill with all smallest coins
	for i:=uint64(0);i< maxCoins;i++{
		cs := os.pop(0)
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
			newSum = sum-funds[i].Value()+os.get(j).Value()
			if newSum>=value{
				break
			}
			j++
		}
		oldSleeve := funds[i]
		funds[i] = os.pop(j)
		os.add(oldSleeve)
		sum = newSum
		if sum>=value{
			goto Success
		}
	}

	// Step 3: If nothing is found, add funds back onto the ordered list,
	// it will be all the highest coins so it can just be appended
	*os.list = append(*os.list,funds...)
	os.value += sum
	os.mutex.Unlock()

	return []coin.Sleeve{},NilSleeve,ErrInvalidOrganizationOfFunds

	Success:

		change := NilSleeve

		if sum>value{
			var err error
			change, err = coin.NewSleeve(sum-value)
			if err!=nil{
				for _,c := range funds{
					os.add(c)
				}
				return []coin.Sleeve{},NilSleeve,err
			}
		}

		os.mutex.Unlock()
		return funds, change, nil
}