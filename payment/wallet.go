////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"gitlab.com/privategrity/client/parse"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/crypto/coin"
	"time"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/client/api"
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

// TODO initialize this? should this be global?
var WalletyMcWalletFace Wallet

func init() {
	// Add incoming invoice listener
	api.Listen(user.ID(0), parse.Type_PAYMENT_INVOICE, &InvoiceListener{})
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

type InvoiceListener struct{}

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
	WalletyMcWalletFace.inboundRequests.Add(msg.Hash(), transaction)
	// and save it
	user.TheSession.StoreSession()
}
