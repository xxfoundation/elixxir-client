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

	session user.Session
}

func CreateWallet(s user.Session) (*Wallet, error) {

	cs, err := CreateOrderedStorage(CoinStorageTag, s)

	if err != nil {
		return nil, err
	}

	obr, err := CreateTransactionList(OutboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	ibr, err := CreateTransactionList(InboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	pt, err := CreateTransactionList(PendingTransactionsTag, s)

	if err != nil {
		return nil, err
	}

	w := &Wallet{coinStorage: cs, outboundRequests: obr,
		inboundRequests: ibr, pendingTransactions: pt}

	return w, nil
}

// You need to call this method after creating the wallet to have the wallet
// behave correctly when receiving messages
// TODO: Should this take the listeners as parameters?
func (w *Wallet) RegisterListeners() {
	w.registerInvoiceListener()
}

func (w *Wallet) registerInvoiceListener() {
	// Add incoming invoice listener
	api.Listen(user.ID(0), parse.Type_PAYMENT_INVOICE, &InvoiceListener{
		wallet: w,
	})
}

// Creates an invoice, which you can add to the wallet and create a message of
// You must call RegisterInvoice after this to put the invoice in the
// queue of outbound requests, so that you can keep the coins in your wallet
// if your payer pays the invoice.
func (w *Wallet) CreateInvoice(moneyFrom user.ID, value uint64,
	memo string) (*Transaction, error) {
	newCoin, err := coin.NewSleeve(value)

	if err != nil {
		return nil, err
	}

	return &Transaction{
		Create:    newCoin,
		Sender:    w.session.GetCurrentUser().UserID,
		Recipient: moneyFrom,
		Value:     value,
		Memo:      memo,
		Timestamp: time.Now(),
	}, nil
}

func (w *Wallet) RegisterInvoice(id parse.MessageHash,
	invoice *Transaction) error {
	w.outboundRequests.Add(id, invoice)
	return w.session.StoreSession()
}

type InvoiceListener struct {
	wallet *Wallet
}

func (il *InvoiceListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	var invoice parse.PaymentInvoice

	// Test for incorrect message type, just in case
	if msg.Type != parse.Type_PAYMENT_INVOICE {
		jww.WARN.Printf("InvoiceListener: Got an invoice with the incorrect"+
			" type: %v",
			msg.Type.String())
		return
	}

	// Don't humor people who send malformed messages
	if err := proto.Unmarshal(msg.Body, &invoice); err != nil {
		jww.WARN.Printf("InvoiceListener: Got error unmarshaling inbound"+
			" invoice: %v", err.Error())
		return
	}

	if uint64(len(invoice.CreatedCoin)) != coin.BaseFrameLen {
		jww.WARN.Printf("InvoiceListener: Created coin has incorrect length"+
			" %v and is likely invalid", len(invoice.CreatedCoin))
		return
	}

	if !coin.IsCompound(invoice.CreatedCoin) {
		jww.WARN.Printf("InvoiceListener: Got an invoice with an incorrect" +
			" coin type")
		return
	}

	// Convert the message to a compound
	var compound coin.Compound
	copy(compound[:], invoice.CreatedCoin)

	transaction := &Transaction{
		Create:    coin.ConstructSleeve(nil, &compound),
		Destroy:   nil,
		Change:    NilSleeve,
		Sender:    msg.Sender,
		Recipient: msg.Receiver,
		Memo:      invoice.Memo,
		Timestamp: time.Unix(invoice.Time, 0),
		Value:     compound.Value(),
	}

	// Actually add the request to the list of inbound requests
	il.wallet.inboundRequests.Add(msg.Hash(), transaction)
	// and save it
	il.wallet.session.StoreSession()
}
