////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"encoding/gob"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
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
func createTransactionList(tag string, session user.Session) (*TransactionList, error) {
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
func (tl *TransactionList) upsert(mh parse.MessageHash, t *Transaction) {
	tl.session.LockStorage()
	tl.upsertImpl(mh, t)
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
func (tl *TransactionList) pop(mh parse.MessageHash) (*Transaction, bool) {
	tl.session.LockStorage()
	t, b := tl.popImpl(mh)
	tl.session.UnlockStorage()
	return t, b
}

// INTERNAL FUNCTIONS

func (tl *TransactionList) upsertImpl(mh parse.MessageHash, t *Transaction) {
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

func (tl *TransactionList) popImpl(mh parse.MessageHash) (*Transaction, bool) {
	t, b := tl.get(mh)
	if b {
		tl.value -= t.Value
		delete(*tl.transactionMap, mh)
	}
	return t, b
}

// TODO Is there actually any reason to return the transaction key here?
// It's useful if you want to pop a transaction from e.g. the invoice list,
// so I think it may as well stay
type KeyAndTransaction struct {
	Key         *parse.MessageHash
	Transaction *Transaction
}

type By func(t1, t2 *Transaction) bool

var ByValue By = func(t1, t2 *Transaction) bool {
	return t1.Value < t2.Value
}

var ByTimestamp By = func(t1, t2 *Transaction) bool {
	return t1.Timestamp.Before(t2.Timestamp)
}

// Implement sort.Interface
type transactionSorter struct {
	// Can you sort the map directly or do you need to copy pointers to a slice
	// first?
	// I guess there's no reason not to make a slice since you'll need one to
	// return anyway. Unless there's some really complicated algorithm you
	// can use to sort faster not in-place, which I think I'd have heard of by
	// now.
	transactions []KeyAndTransaction
	by           By
}

func (s *transactionSorter) Len() int {
	return len(s.transactions)
}

func (s *transactionSorter) Swap(i, j int) {
	s.transactions[i], s.transactions[j] = s.transactions[j], s.transactions[i]
}

func (s *transactionSorter) Less(i, j int) bool {
	return s.by(s.transactions[i].Transaction, s.transactions[j].Transaction)
}

// The created struct will hold all of the information needed to sort the
// transaction list in a certain way
func (tl *TransactionList) createTransactionSorter(by By) transactionSorter {
	transactions := make([]KeyAndTransaction, 0, len(*tl.transactionMap))
	for k, v := range *tl.transactionMap {
		transactions = append(transactions, KeyAndTransaction{
			Key:         &k,
			Transaction: v,
		})
	}
	result := transactionSorter{
		transactions: transactions,
		by:           by,
	}
	return result
}

// Returns a snapshot of a transaction list at a certain point in time.
// Getting this could be expensive if there are enough transactions in the list!
func (tl *TransactionList) GetTransactionView(by By, reverse bool) []KeyAndTransaction {
	tl.session.LockStorage()
	sorter := tl.createTransactionSorter(by)
	if reverse {
		sort.Sort(sort.Reverse(&sorter))
	} else {
		sort.Sort(&sorter)
	}
	tl.session.UnlockStorage()
	return sorter.transactions
}
