////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/parse"
	"errors"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/payment"
	"gitlab.com/elixxir/primitives/userid"
)

// Currently there's only one wallet that you can get
// There may be many in the future
func GetActiveWallet() *Wallet {
	return &Wallet{wallet: api.Wallet()}
}

func (w *Wallet) Listen(userId []byte, messageType int32, newListener Listener) string {
	typedUserId := new(userid.UserID).SetBytes(userId)

	listener := &listenerProxy{proxy: newListener}

	return api.Listen(typedUserId, cmixproto.Type(messageType), listener,
		w.wallet.GetSwitchboard())
}

func (w *Wallet) StopListening(listenerHandle string) {
	api.StopListening(listenerHandle, w.wallet.GetSwitchboard())
}

// Returns the currently available balance in the wallet
func (w *Wallet) GetAvailableFunds() int64 {
	return int64(api.Wallet().GetAvailableFunds())
}

// Payer: user ID, 256 bits
// Value: must be positive
// Send the returned message unless you get an error
func (w *Wallet) Invoice(payer []byte, value int64, memo string) (Message, error) {
	userId := new(userid.UserID).SetBytes(payer)
	msg, err := w.wallet.Invoice(userId, value, memo)
	return &parse.BindingsMessageProxy{Proxy: msg}, err
}

// Get an invoice handle by listening to the wallet's PAYMENT_INVOICE_UI
// messages
// Returns a payment message that the bindings user can send at any time
// they wish
func (w *Wallet) Pay(invoiceHandle []byte) (Message, error) {
	var typedInvoiceId parse.MessageHash
	copiedLen := copy(typedInvoiceId[:], invoiceHandle)
	if copiedLen != parse.MessageHashLen {
		return nil, errors.New(fmt.Sprintf("Invoice ID wasn't long enough. "+
			"Got %v bytes, needed %v bytes.", copiedLen, parse.MessageHashLen))
	}

	msg, err := w.wallet.Pay(typedInvoiceId)
	if err != nil {
		return nil, err
	}
	proxyMsg := parse.BindingsMessageProxy{Proxy: msg}
	return &proxyMsg, nil
	// After receiving a response from the payment bot,
	// the wallet will automatically send a receipt to the original sender of
	// the invoice. Make sure to listen for PAYMENT_RECEIPT_UI messages to
	// be able to handle the receipt.
}

// Proxy structure that provides wallet functionality
// Bindings authors: When trying to call methods on the underlying wallet,
// please be careful to not recurse infinitely by accidentally calling the
// methods on the proxy structure:
// func (w *Wallet) Pay() { ...w.wallet.Pay() } over
// func (w *Wallet) Pay() { ...w.Pay()... }
type Wallet struct {
	wallet *payment.Wallet
}

// Meant to be read-only in the current version of the API
type Transaction struct {
	// This is the ID that you can use to look up this transaction in the map
	// Of course, since you already have it if this field is populated, I'm not
	// sure how useful it is. Candidate for removal.
	ID []byte
	// What the transaction is for
	// e.g. "hardware", "coffee getting"
	Memo string
	// Time the transaction originated (UTC seconds since Unix epoch)
	Timestamp int64
	// Number of tokens transferred
	Value int64
	// ID of the invoice that initiated this transaction
	// May be same as the ID field
	// 256 bits long
	InvoiceID []byte
	// ID of the user who sends the tokens in the transaction
	// 256 bits long
	SenderID []byte
	// ID of the user who receives the tokens in the transaction
	// 256 bits long
	ReceiverID []byte
	// TBD: Expose created/destroyed/change? Images only to avoid revealing
	// secrets to the UI? For now I'm leaving it out.
}

// Bindings to a slice of *Transaction
// To get the total value of the transaction list, iterate through it and sum
// the value of each transaction.
type TransactionView struct {
	transactions []*Transaction
}

func (t *TransactionView) Get(index int) *Transaction {
	return t.transactions[index]
}

func (t *TransactionView) Length() int {
	return len(t.transactions)
}

func createTransaction(transaction payment.KeyAndTransaction) *Transaction {
	// TBD: Do a deep copy to ensure that changing the fields doesn't change
	// the state of the underlying wallet?
	return &Transaction{
		ID:        transaction.Key[:],
		Memo:      transaction.Transaction.Memo,
		Timestamp: transaction.Transaction.Timestamp.Unix(),
		// this cast seems bad
		Value:      int64(transaction.Transaction.Value),
		InvoiceID:  transaction.Transaction.OriginID[:],
		SenderID:   transaction.Transaction.Sender.Bytes(),
		ReceiverID: transaction.Transaction.Recipient.Bytes(),
	}
}

func createTransactionView(transactions []payment.KeyAndTransaction) *TransactionView {
	result := TransactionView{
		transactions: make([]*Transaction, len(transactions)),
	}
	for i := range transactions {
		result.transactions[i] = createTransaction(transactions[i])
	}
	return nil
}

func (t *TransactionList) CreateTransactionView(order int32) *TransactionView {
	switch cmixproto.TransactionListOrder(order) {
	case cmixproto.TransactionListOrder_TIMESTAMP_DESCENDING:
		return createTransactionView(t.tList.GetTransactionView(payment.
			ByTimestamp, true))
	case cmixproto.TransactionListOrder_TIMESTAMP_ASCENDING:
		return createTransactionView(t.tList.GetTransactionView(payment.
			ByTimestamp, false))
	case cmixproto.TransactionListOrder_VALUE_DESCENDING:
		return createTransactionView(t.tList.GetTransactionView(payment.ByValue,
			true))
	case cmixproto.TransactionListOrder_VALUE_ASCENDING:
		return createTransactionView(t.tList.GetTransactionView(payment.ByValue,
			false))
	}
	// TBD return error instead of failing silently?
	return nil
}

type TransactionList struct {
	tList *payment.TransactionList
}

// There's a possibility that if you wait too long,
// someone will pay this transaction and it will move to a different list,
// making your key data stale. So, please be careful and handle all errors.
func (t *TransactionList) Get(key []byte) (*Transaction, error) {
	var transactionId parse.MessageHash
	copy(transactionId[:], key)
	transaction, ok := t.tList.Get(transactionId)
	if !ok {
		return nil, fmt.Errorf("Couldn't get transaction with key %q", key)
	} else {
		return createTransaction(payment.KeyAndTransaction{
			Key:         &transactionId,
			Transaction: transaction,
		}), nil
	}
}

func (t *TransactionList) Value() int64 {
	// this cast seems bad
	return int64(t.tList.Value())
}

func (w *Wallet) GetInboundRequests() *TransactionList {
	return &TransactionList{tList: w.wallet.GetInboundRequests()}
}
func (w *Wallet) GetOutboundRequests() *TransactionList {
	return &TransactionList{tList: w.wallet.GetOutboundRequests()}
}
func (w *Wallet) GetPendingTransactions() *TransactionList {
	return &TransactionList{tList: w.wallet.GetPendingTransactions()}
}
func (w *Wallet) GetCompleteInboundPayments() *TransactionList {
	return &TransactionList{tList: w.wallet.GetCompletedInboundPayments()}
}
func (w *Wallet) GetCompletedOutboundPayments() *TransactionList {
	return &TransactionList{tList: w.wallet.GetCompletedOutboundPayments()}
}
