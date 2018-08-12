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
	"sort"
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
	// FIXME for an Upsert the recalculation of the value isn't technically
	// correct. this only matters if you upsert the same hash more than once.
	// The easiest fix is to iterate the whole map and update the value cache
	// each time, or to have Value() just sum up all the entries in the map.
	tl.value += t.Value
}

func (tl *TransactionList) get(mh parse.MessageHash) (*Transaction, bool) {
	t, b := (*tl.transactionMap)[mh]
	return t, b
}

func (tl *TransactionList) pop(mh parse.MessageHash) (*Transaction, bool) {
	t, b := tl.get(mh)
	if b {
		tl.value -= t.Value
		delete(*tl.transactionMap, mh)
	}
	return t, b
}

// Used for sorting transaction lists by their various fields.
// For bindings reasons, we can't return sorted slices of transactions, so
// when sorting the transaction map we need to return the keys that go with
// the sorted transaction
type keyAndTransaction struct {
	key         parse.MessageHash
	transaction *Transaction
}

// TODO Expose to API? There are a lot of variants...
// Golang randomizes map keys' order when you range through a map.
// To avoid showing transactions to our users in that random order, we have to
// sort the map's keys by values in the transactions.
// Useful sorting criteria are timestamp and value (possibly also grouped by the
// other party to the transaction.)
func (tl *TransactionList) GetKeysByTimestampDescending() []byte {
	return tl.getKeyList(func(t1, t2 *Transaction) bool {
		return t1.Timestamp.After(t2.Timestamp)
	})
}

// Sorts the keys in a transaction list by a function comparing two transactions
func (tl *TransactionList) getKeyList(lessThan func(t1, t2 *Transaction) bool) []byte {
	tl.session.LockStorage()
	keys := make([]keyAndTransaction, 0, len(*tl.transactionMap))
	for k, v := range *tl.transactionMap {
		keys = append(keys, keyAndTransaction{
			key:         k,
			transaction: v,
		})
	}
	// Sort the keys in descending order by timestamp
	sort.Slice(keys, func(i, j int) bool {
		return lessThan(keys[i].transaction, keys[j].transaction)
	})

	keyList := make([]byte, 0, uint64(len(*tl.transactionMap))*parse.
		MessageHashLen)
	tl.session.UnlockStorage()
	for i := range keys {
		keyList = append(keyList, keys[i].key[:]...)
	}
	return keyList
}
