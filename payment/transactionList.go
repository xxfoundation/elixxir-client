package payment

import (
	"sync"
	"gitlab.com/privategrity/client/parse"
	"encoding/gob"
	"gitlab.com/privategrity/client/user"
)

type TransactionList struct{
	transactionMap *map[parse.MessageHash]*Transaction
	mutex sync.Mutex
	value uint64
}

func NewTransactionList(tag string)(*TransactionList,error){
	gob.Register(TransactionList{})

	var tlmPtr *map[parse.MessageHash]*Transaction

	tli, err := user.TheSession.QueryMap(tag)

	if err!=nil{
		//If there is an err make the object
		tlMap := make(map[parse.MessageHash]*Transaction)
		tlmPtr = &tlMap

		if err == user.ErrQuery {
			err = user.TheSession.UpsertMap(tag, tlmPtr)
		}
		if err != nil {
			return nil, err
		}
	}else{
		tlmPtr = tli.(*map[parse.MessageHash]*Transaction)
	}

	value := uint64(0)

	for _, t := range *tlmPtr{
		value += t.Value
	}

	return &TransactionList{transactionMap:tlmPtr,value:value}, nil
}

func (tl *TransactionList) Value()uint64{
	tl.mutex.Lock()
	v := tl.value
	tl.mutex.Unlock()
	return v
}

func (tl *TransactionList) add(mh parse.MessageHash, t *Transaction){
	(*tl.transactionMap)[mh] = t
	tl.value += tl.Value()
}

func (tl *TransactionList) Add(mh parse.MessageHash, t *Transaction){
	tl.mutex.Lock()
	tl.add(mh, t)
	tl.mutex.Unlock()
}

func (tl *TransactionList) get(mh parse.MessageHash)(*Transaction, bool){
	t, b := (*tl.transactionMap)[mh]
	return t, b
}

func (tl *TransactionList) Get(mh parse.MessageHash)(*Transaction, bool){
	tl.mutex.Lock()
	t, b := tl.get(mh)
	tl.mutex.Unlock()
	return t,b
}
