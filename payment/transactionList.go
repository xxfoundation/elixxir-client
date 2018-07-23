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
	transactionMap map[parse.MessageHash]*Transaction
	value          uint64

	session user.Session
}

func CreateTransactionList(tag string, session user.Session) (*TransactionList, error) {
	gob.Register(TransactionList{})
	gob.Register(make(map[parse.MessageHash]*Transaction))

	var tlm map[parse.MessageHash]*Transaction

	tli, err := session.QueryMap(tag)

	if err != nil {
		//If there is an err make the object
		tlm = make(map[parse.MessageHash]*Transaction)

		if err == user.ErrQuery {
			err = session.UpsertMap(tag, tlm)
		}
		if err != nil {
			return nil, err
		}
	} else {
		tlm = tli.(map[parse.MessageHash]*Transaction)
	}

	value := uint64(0)

	for _, t := range tlm {
		value += t.Value
	}

	return &TransactionList{transactionMap: tlm, value: value, session: session}, nil
}

func (tl *TransactionList) Value() uint64 {
	tl.session.LockStorage()
	v := tl.value
	tl.session.UnlockStorage()
	return v
}

func (tl *TransactionList) add(mh parse.MessageHash, t *Transaction) {
	tl.transactionMap[mh] = t
	tl.value += t.Value
}

func (tl *TransactionList) Add(mh parse.MessageHash, t *Transaction) {
	tl.session.LockStorage()
	tl.add(mh, t)
	tl.session.UnlockStorage()
}

func (tl *TransactionList) get(mh parse.MessageHash) (*Transaction, bool) {
	t, b := tl.transactionMap[mh]
	return t, b
}

func (tl *TransactionList) Get(mh parse.MessageHash) (*Transaction, bool) {
	tl.session.LockStorage()
	t, b := tl.get(mh)
	tl.session.UnlockStorage()
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
