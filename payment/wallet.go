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

func (t *Transaction) FormatInvoice() (*parse.Message, error) {
	invoice := parse.PaymentInvoice{
		Time:         time.Now().Unix(),
		CreatedCoins: t.Create.Compound()[:],
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
		// TODO populate nonce and panic if outgoing message doesn't have one
		Nonce: nil,
	}, nil
}
