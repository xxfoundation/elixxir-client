////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"encoding/gob"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
)

type TransactionList struct {
	transactionMap *map[parse.MessageHash]*Transaction
	value          uint64

	session user.Session
}

func init() {
	m := make(map[parse.MessageHash]*Transaction)
	gob.Register(&m)
}

// Checks to see if a transaction list of the given tag is present in session.  If one is, then it returns it.
// If one isn't, then a new one is created
func CreateTransactionList(tag string, session user.Session) (*TransactionList, error) {
	gob.Register(TransactionList{})

	var tlmPtr *map[parse.MessageHash]*Transaction

	tli, err := session.QueryMap(tag)

	if err != nil {
		//If there is an err make the object
		tlMap := make(map[parse.MessageHash]*Transaction)
		tlmPtr = &tlMap

		if err == user.ErrQuery {
			err = session.UpsertMap(tag, tlmPtr)
		}
		if err != nil {
			return nil, err
		}
	} else {
		tlmPtr = tli.(*map[parse.MessageHash]*Transaction)
	}

	value := uint64(0)

	for _, t := range *tlmPtr {
		value += t.Value
	}

	return &TransactionList{transactionMap: tlmPtr, value: value, session: session}, nil
}

// Returns the value of all transactions in the list
func (tl *TransactionList) Value() uint64 {
	tl.session.LockStorage()
	v := tl.value
	tl.session.UnlockStorage()
	return v
}

// Adds or updates a transaction to the list with a key of the given hash
func (tl *TransactionList) Upsert(mh parse.MessageHash, t *Transaction) {
	tl.session.LockStorage()
	tl.upsert(mh, t)
	tl.session.UnlockStorage()
}

// Gets a transaction from the list with a key of the given hash
func (tl *TransactionList) Get(mh parse.MessageHash) (*Transaction, bool) {
	tl.session.LockStorage()
	t, b := tl.get(mh)
	tl.session.UnlockStorage()
	return t, b
}

// Pops a transaction from the list with a key of the given hash
func (tl *TransactionList) Pop(mh parse.MessageHash) (*Transaction, bool) {
	tl.session.LockStorage()
	t, b := tl.pop(mh)
	tl.session.UnlockStorage()
	return t, b
}

// INTERNAL FUNCTIONS

func (tl *TransactionList) upsert(mh parse.MessageHash, t *Transaction) {
	(*tl.transactionMap)[mh] = t
	tl.value += t.Value
}

func (tl *TransactionList) get(mh parse.MessageHash) (*Transaction, bool) {
	t, b := (*tl.transactionMap)[mh]
	return t, b
}

func (tl *TransactionList) pop(mh parse.MessageHash) (*Transaction, bool) {
	t, b := tl.get(mh)
	if b {
		delete(*tl.transactionMap, mh)
	}
	return t, b
}

func (tl *TransactionList) pop(mh parse.MessageHash) (*Transaction, bool) {
	t, b := tl.get(mh)
	if b {
		delete(tl.transactionMap, mh)
	}
	return t, b
}

func (tl *TransactionList) Pop(mh parse.MessageHash) (*Transaction, bool) {
	tl.session.LockStorage()
	t, b := tl.pop(mh)
	tl.session.UnlockStorage()
	return t, b
}
