////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"errors"
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/crypto/format"
	"time"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/api"
)

const CoinStorageTag = "CoinStorage"
const OutboundRequestsTag = "OutboundRequests"
const InboundRequestsTag = "InboundRequests"
const PendingTransactionsTag = "PendingTransactions"
const InboundPaymentsTag = "InboundPayments"
const OutboundPaymentsTag = "OutboundPayments"

type Wallet struct {
	// Stores the user's compound coins
	coinStorage *OrderedCoinStorage
	// Invoices this user made to another user
	outboundRequests *TransactionList
	// Invoices this user got from another user
	inboundRequests *TransactionList
	// Transactions that are in processing on the payment bot
	pendingTransactions *TransactionList
	// Completed payments to the user
	inboundPayments *TransactionList
	// Completed payments from the user
	outboundPayments *TransactionList

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

	ip, err := CreateTransactionList(InboundPaymentsTag, s)

	if err != nil {
		return nil, err
	}

	op, err := CreateTransactionList(OutboundPaymentsTag, s)

	if err != nil {
		return nil, err
	}

	w := &Wallet{
		coinStorage:         cs,
		outboundRequests:    obr,
		inboundRequests:     ibr,
		pendingTransactions: pt,
		inboundPayments:     ip,
		outboundPayments:    op,
		session:             s}

	w.RegisterListeners()

	return w, nil
}

// You need to call this method after creating the wallet to have the wallet
// behave correctly when receiving messages
// TODO: Should this take the listeners as parameters?
func (w *Wallet) RegisterListeners() {
	switchboard.Listeners.Register(user.ID(0), parse.Type_PAYMENT_INVOICE, &InvoiceListener{
		wallet: w,
	})
	switchboard.Listeners.Register(getPaymentBotID(), parse.Type_PAYMENT_RESPONSE, &ResponseListener{
		wallet: w,
	})
}

// Creates an invoice, which you can add to the wallet and create a message of
func createInvoice(payer user.ID, payee user.ID, value uint64,
	memo string) (*Transaction, error) {
	newCoin, err := coin.NewSleeve(value)

	if err != nil {
		return nil, err
	}

	// TODO Are the payer and payee in the correct fields?
	return &Transaction{
		Create:    newCoin,
		Sender:    payer,
		Recipient: payee,
		Value:     value,
		Memo:      memo,
		Timestamp: time.Now(),
	}, nil
}

// Registers an invoice with the session and wallet
func (w *Wallet) registerInvoice(id parse.MessageHash,
	invoice *Transaction) error {
	w.outboundRequests.Upsert(id, invoice)
	return w.session.StoreSession()
}

// Creates, formats, and registers an invoice in the outgoing requests
// Assumes that the payee is the current user in the session
func (w *Wallet) Invoice(payer user.ID, value uint64,
	memo string) (*parse.Message, error) {
	transaction, err := createInvoice(payer, w.session.GetCurrentUser().UserID,
		value, memo)
	if err != nil {
		return nil, err
	}
	msg := transaction.FormatPaymentInvoice()
	w.registerInvoice(msg.Hash(), transaction)
	return msg, nil
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
		Sender:    msg.Receiver,
		Recipient: msg.Sender,
		Memo:      invoice.Memo,
		Timestamp: time.Unix(invoice.Time, 0),
		Value:     compound.Value(),
	}

	// Actually add the request to the list of inbound requests
	il.wallet.inboundRequests.Upsert(msg.Hash(), transaction)
	// and save it
	il.wallet.session.StoreSession()

	// Build the message to send the necessary information to the UI
	// It's a list of fixed-length transaction IDs that the UI can use to get
	// the transactions in the list in time order.
	// This way, if the UI needs ten photos, it doesn't need to get and
	// deserialize a hundred complete invoices all in one protobuf.
	// It can just get the transaction information that it needs.
	keyList := il.wallet.inboundRequests.GetKeysByTimestampDescending()
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_INVOICE_UI,
			Body: keyList,
		},
		Sender:   getPaymentBotID(),
		Receiver: 0,
		Nonce:    nil,
	})
}

func getPaymentBotID() user.ID {
	return 17
}

func buildPaymentPayload(request, change coin.Sleeve,
	funds []coin.Sleeve) []byte {
	// The order of these doesn't matter because the coin's header determines
	// whether you are funding or destroying the coin on the payment bot.
	payload := make([]byte, 0, format.DATA_LEN)
	// So, I'll just use an arbitrary order. The invoiced coin can go first.
	payload = append(payload, request.Compound()[:]...)
	// Then, the change, if there is any
	if change != NilSleeve {
		payload = append(payload, change.Compound()[:]...)
	}
	// The funding coins can go next
	for i := range funds {
		payload = append(payload, funds[i].Seed()[:]...)
	}

	return payload
}

func (w *Wallet) Pay(requestID parse.MessageHash) (*parse.Message, error) {
	transaction, ok := w.inboundRequests.Pop(requestID)
	if !ok {
		return nil, errors.New("that request wasn't in the list of inbound" +
			" requests")
	}
	msg, err := w.pay(transaction)
	if err != nil {
		// Roll back the popping
		w.inboundRequests.Upsert(requestID, transaction)
		return nil, err
	}
	errStore := w.session.StoreSession()
	if errStore != nil {
		// Roll back the popping
		w.inboundRequests.Upsert(requestID, transaction)
		return nil, err
	}
	return msg, nil
}

// Create a payment message and register the outgoing payment on pending
// transactions
// TODO As written, the caller is responsible for popping the inbound request
func (w *Wallet) pay(inboundRequest *Transaction) (*parse.Message, error) {
	// Fund from ordered coin storage
	// TODO calculate max coins programmatically? depends on wallet state
	// because change may or may not be present
	funds, change, err := w.coinStorage.Fund(inboundRequest.Value, 4)
	if err != nil {
		return nil, err
	}

	paymentMessage := buildPaymentPayload(inboundRequest.Create, change, funds)

	if uint64(len(parse.Type_PAYMENT_TRANSACTION.Bytes())) + uint64(len(
		paymentMessage)) > format.DATA_LEN {
		// The message is too long to fit in a single payment message
		panic("Payment message doesn't fit in a single message")
	}

	msg := parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_TRANSACTION,
			Body: paymentMessage,
		},
		Sender:   w.session.GetCurrentUser().UserID,
		Receiver: getPaymentBotID(),
		// TODO panic on blank nonce
		Nonce: nil,
	}

	// Register the transaction on the list of outbound transactions
	pendingTransaction := Transaction{
		Create:    inboundRequest.Create,
		Destroy:   funds,
		Change:    change,
		Sender:    inboundRequest.Sender,
		Recipient: inboundRequest.Recipient,
		Memo:      inboundRequest.Memo,
		Timestamp: inboundRequest.Timestamp,
		Value:     inboundRequest.Value,
	}

	w.pendingTransactions.Upsert(msg.Hash(), &pendingTransaction)

	// Return the result.
	return &msg, nil
}

type ResponseListener struct {
	wallet *Wallet
}

func (l *ResponseListener) Hear(msg *parse.Message,
	isHeardElsewhere bool) {
	var response parse.PaymentResponse
	err := proto.Unmarshal(msg.Body, &response)
	if err != nil {
		jww.WARN.Printf("Heard an invalid response from the payment bot. "+
			"Error: %v", err.Error())
	}

	var invoiceID parse.MessageHash
	copy(invoiceID[:], response.ID)

	if !response.Success {
		transaction, ok := l.wallet.pendingTransactions.Pop(invoiceID)
		if !ok {
			jww.WARN.Printf("Couldn't find the transaction with that invoice"+
				" ID: %q", invoiceID)
		} else {
			// Move the coins from pending transactions back to the wallet
			// for now.
			// This may not always be correct - for example, if the coins aren't on
			// the payment bot they might need to be removed from user's wallet
			// so they don't get nothing but declined transactions in the event
			// of corruption.
			for i := range transaction.Destroy {
				l.wallet.coinStorage.Add(transaction.Destroy[i])
			}
		}
	} else {
		// Does it make sense to have the payment bot send the value of the
		// transaction as a response for some quick and dirty verification?

		// Transaction was successful, so remove pending from the wallet
		transaction, ok := l.wallet.pendingTransactions.Pop(invoiceID)
		if !ok {
			jww.WARN.Printf("ResponseListener: Couldn't find the"+
				" transaction with that invoice ID: %q",
				invoiceID)
		} else {
			if transaction.Change != NilSleeve {
				l.wallet.coinStorage.Add(transaction.Change)
			}
			// Send receipt: Need ID of original invoice corresponding to this
			// transaction. That's something that the invoicing client should be
			// able to keep track of.
			l.wallet.outboundPayments.Upsert(invoiceID, transaction)
			receipt := formatReceipt(invoiceID, transaction)
			api.Send(receipt)
		}
	}
	jww.DEBUG.Printf("Payment response: %v", response.Response)
}

func formatReceipt(invoiceID parse.MessageHash, transaction *Transaction) *parse.Message {
	return &parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_RECEIPT,
			Body: invoiceID[:],
		},
		Sender:   user.TheSession.GetCurrentUser().UserID,
		Receiver: transaction.Recipient,
		Nonce:    nil,
	}
}

type ReceiptListener struct {
	wallet *Wallet
}

func (rl *ReceiptListener) Hear(msg *parse.MessageHash, isHeardElsewhere bool) {
	// UI needs to show that the transaction was paid _before_ we remove it
}

func (w *Wallet) GetAvailableFunds() uint64 {
	return w.coinStorage.Value()
}

// Returns a copy of the transaction to keep UIs from changing transaction
func (w *Wallet) GetInboundRequest(id parse.MessageHash) (Transaction, bool) {
	transaction, ok := w.inboundRequests.Get(id)
	// Need to check ok to avoid dereferencing nil transaction
	if !ok {
		return Transaction{}, ok
	} else {
		return *transaction, ok
	}
}

func (w *Wallet) GetOutboundRequest(id parse.MessageHash) (Transaction, bool) {
	transaction, ok := w.outboundRequests.Get(id)
	if !ok {
		return Transaction{}, ok
	} else {
		return *transaction, ok
	}
}

func (w *Wallet) GetPendingTransaction(id parse.MessageHash) (Transaction, bool) {
	transaction, ok := w.pendingTransactions.Get(id)
	if !ok {
		return Transaction{}, ok
	} else {
		return *transaction, ok
	}
}

// Used for adding testing funds to the wallet
func (w *Wallet) Add(funds []coin.Sleeve) {
	for i := range funds {
		w.coinStorage.Add(funds[i])
	}
}
