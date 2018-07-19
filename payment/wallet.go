////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"time"
	"gitlab.com/privategrity/client/api"
)

const CoinStorageTag = "CoinStorage"
const OutboundRequestsTag = "OutboundRequests"
const InboundRequestsTag = "InboundRequests"
const PendingTransactionsTag = "PendingTransactions"

type Wallet struct {
	coinStorage         *OrderedCoinStorage
	outboundRequests    *TransactionList
	inboundRequests     *TransactionList
	pendingTransactions *TransactionList
}

// Modify new wallet so that when it is called a bunch of listeners have to be passed
func NewWallet() (*Wallet, error) {

	cs, err := NewOrderedStorage(CoinStorageTag)

	if err != nil {
		return nil, err
	}

	obr, err := NewTransactionList(OutboundRequestsTag)

	if err != nil {
		return nil, err
	}

	ibr, err := NewTransactionList(InboundRequestsTag)

	if err != nil {
		return nil, err
	}

	pt, err := NewTransactionList(PendingTransactionsTag)

	if err != nil {
		return nil, err
	}

	w := &Wallet{coinStorage: cs, outboundRequests: obr,
	inboundRequests: ibr, pendingTransactions: pt}

	// Add incoming invoice listener
	api.Listen(user.ID(0), parse.Type_PAYMENT_INVOICE, &InvoiceListener{
		wallet: w,
	})

	return w, nil
}

// Adds a fund request to the wallet and returns the message to make it
func (w *Wallet) Invoice(from user.ID, value uint64, description string) (*parse.Message, error) {

	newCoin, err := coin.NewSleeve(value)

	if err != nil {
		return nil, err
	}

	invoiceTransaction := Transaction{
		Create:    newCoin,
		Sender:    user.TheSession.GetCurrentUser().UserID,
		Recipient: from,
		Value:     value,
	}

	invoiceMessage, err := invoiceTransaction.FormatInvoice()

	if err != nil {
		return nil, err
	}

	invoiceHash := invoiceMessage.Hash()

	w.outboundRequests.Add(invoiceHash, &invoiceTransaction)

	user.TheSession.StoreSession()

	return invoiceMessage, nil
}

type InvoiceListener struct{
	wallet *Wallet
}

func (il *InvoiceListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	var invoice parse.PaymentInvoice

	// Don't humor people who send malformed messages
	if err := proto.Unmarshal(msg.Body, &invoice); err != nil {
		jww.WARN.Printf("Got error unmarshaling inbound invoice: %v", err.Error())
		return
	}

	if !coin.IsCompound(invoice.CreatedCoin) {
		jww.WARN.Printf("Got an invoice with an incorrect coin type")
		return
	}

	// Convert the message to a compound
	var compound coin.Compound
	copy(compound[:], invoice.CreatedCoin)

	transaction := &Transaction{
		Create:      coin.ConstructSleeve(nil, &compound),
		Destroy:     nil,
		Change:      NilSleeve,
		Sender:      msg.Sender,
		Recipient:   msg.Receiver,
		Description: invoice.Memo,
		Timestamp:   time.Unix(invoice.Time, 0),
		Value:       compound.Value(),
	}

	// Actually add the request to the list of inbound requests
	il.wallet.inboundRequests.Add(msg.Hash(), transaction)
	// and save it
	user.TheSession.StoreSession()
}
