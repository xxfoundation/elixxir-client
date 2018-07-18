////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"gitlab.com/privategrity/client/parse"
	"time"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/crypto/coin"
	jww "github.com/spf13/jwalterweatherman"
)

const CoinStorageTag string = "CoinStorage"
const OutboundRequestsTag string = "OutboundRequests"
const InboundRequestsTag string = "InboundRequests"
const PendingTransactionsTag string = "PendingTransactions"

type Wallet struct {
	coinStorage         *OrderedCoinStorage
	outboundRequests    *TransactionList
	inboundRequests     *TransactionList
	pendingTransactions *TransactionList
}

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

	return &Wallet{coinStorage: cs, outboundRequests: obr, inboundRequests: ibr, pendingTransactions: pt}, nil
}

// FIXME Limit this to one part message (requires message ID revamp for accuracy)
func (t *Transaction) FormatInvoice() (*parse.Message, error) {
	compound := t.Create.Compound()
	invoice := parse.PaymentInvoice{
		Time:         time.Now().Unix(),
		CreatedCoins: compound[:],
		Memo:         t.Description,
	}
	wireRep, err := proto.Marshal(&invoice)
	if err != nil {
		return nil, err
	}

	typedBody := parse.TypedBody{
		Type: parse.Type_PAYMENT_INVOICE,
		Body: wireRep,
	}

	return &parse.Message{
		TypedBody: typedBody,
		Sender:    t.Sender,
		Receiver:  t.Recipient,
		// TODO populate nonce and panic if any outgoing message has none
		Nonce: nil,
	}, nil
}

func (w *Wallet) HearInboundRequest(msg *parse.Message) {
	var invoice parse.PaymentInvoice

	// Don't humor people who send malformed messages
	if err := proto.Unmarshal(msg.Body, &invoice); err != nil {
		jww.WARN.Printf("Got error unmarshaling inbound invoice: %v", err.Error())
		return
	}

	if !coin.IsCompound(invoice.CreatedCoins) {
		jww.WARN.Printf("Got an invoice with an incorrect coin type")
		return
	}

	// Convert the message to a compound
	var compound coin.Compound
	copy(compound[:], invoice.CreatedCoins)

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

	w.inboundRequests.Add(msg.Hash(), transaction)
}

// This listener hears incoming invoices and puts them on the list
// TODO hook the wallet up to the listeners somewhere
func (w *Wallet) Hear(msg *parse.Message, isHeardElsewhere bool) {
	switch msg.Type {
	case parse.Type_PAYMENT_INVOICE:
		w.HearInboundRequest(msg)
		break
	}
}
