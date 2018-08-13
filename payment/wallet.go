////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"encoding/base64"
	"errors"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/crypto/format"
	"time"
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
	completedInboundPayments *TransactionList
	// Completed payments from the user
	completedOutboundPayments *TransactionList

	session user.Session
}

func CreateWallet(s user.Session, doMint bool) (*Wallet, error) {

	cs, err := CreateOrderedStorage(CoinStorageTag, s)

	if err != nil {
		return nil, err
	}

	if doMint {
		mintedCoins := coin.MintUser(uint64(s.GetCurrentUser().UserID))
		for i := range mintedCoins {
			cs.add(mintedCoins[i])
		}
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
		coinStorage:               cs,
		outboundRequests:          obr,
		inboundRequests:           ibr,
		pendingTransactions:       pt,
		completedInboundPayments:  ip,
		completedOutboundPayments: op,
		session:                   s}

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
	switchboard.Listeners.Register(user.ID(0), parse.Type_PAYMENT_RECEIPT, &ReceiptListener{
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
func (w *Wallet) registerInvoice(invoice *Transaction) error {
	w.outboundRequests.Upsert(invoice.OriginID, invoice)
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
	transaction.OriginID = msg.Hash()
	w.registerInvoice(transaction)
	return msg, nil
}

type InvoiceListener struct {
	wallet *Wallet
}

func (il *InvoiceListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	var invoice parse.PaymentInvoice

	// Test for incorrect message type, just in case
	if msg.Type != parse.Type_PAYMENT_INVOICE {
		globals.N.WARN.Printf("InvoiceListener: Got an invoice with the incorrect"+
			" type: %v",
			msg.Type.String())
		return
	}

	// Don't humor people who send malformed messages
	if err := proto.Unmarshal(msg.Body, &invoice); err != nil {
		globals.N.WARN.Printf("InvoiceListener: Got error unmarshaling inbound"+
			" invoice: %v", err.Error())
		return
	}

	if uint64(len(invoice.CreatedCoin)) != coin.BaseFrameLen {
		globals.N.WARN.Printf("InvoiceListener: Created coin has incorrect length"+
			" %v and is likely invalid", len(invoice.CreatedCoin))
		return
	}

	if !coin.IsCompound(invoice.CreatedCoin) {
		globals.N.WARN.Printf("InvoiceListener: Got an invoice with an incorrect" +
			" coin type")
		return
	}

	// Convert the message to a compound
	var compound coin.Compound
	copy(compound[:], invoice.CreatedCoin)

	invoiceID := msg.Hash()
	transaction := &Transaction{
		Create:    coin.ConstructSleeve(nil, &compound),
		Sender:    msg.Receiver,
		Recipient: msg.Sender,
		Memo:      invoice.Memo,
		Timestamp: time.Unix(invoice.Time, 0),
		Value:     compound.Value(),
		OriginID:  invoiceID,
	}

	// Actually add the request to the list of inbound requests
	il.wallet.inboundRequests.Upsert(invoiceID, transaction)
	// and save it
	il.wallet.session.StoreSession()

	// The invoice UI message allows the UI to notify the user that the new
	// invoice is here and ready to be paid
	switchboard.Listeners.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_INVOICE_UI,
			Body: invoiceID[:],
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
		OriginID:  inboundRequest.OriginID,
	}

	paymentID := msg.Hash()
	globals.N.INFO.Printf("Prepared payment message. Its ID is %v",
		base64.StdEncoding.EncodeToString(paymentID[:]))
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
		globals.N.WARN.Printf("Heard an invalid response from the payment bot. "+
			"Error: %v", err.Error())
	}

	var paymentID parse.MessageHash
	copy(paymentID[:], response.ID)
	globals.N.INFO.Printf("Heard response from payment bot. ID: %v",
		base64.StdEncoding.EncodeToString(paymentID[:]))
	transaction, ok := l.wallet.pendingTransactions.Pop(paymentID)
	if !ok {
		globals.N.ERROR.Printf("Couldn't find the transaction with that"+
			" payment message ID: %q", paymentID)
		return
	}

	if !response.Success {
		// Move the coins from pending transactions back to the wallet
		// for now.
		// This may not always be correct - for example, if the coins
		// aren't on the payment bot they might need to be removed from
		// user's wallet so they don't get nothing but declined
		// transactions in the event of corruption.
		for i := range transaction.Destroy {
			l.wallet.coinStorage.Add(transaction.Destroy[i])
		}
	} else {
		// Does it make sense to have the payment bot send the value of the
		// transaction as a response for some quick and dirty verification?

		// Transaction was successful, so remove pending from the wallet
		if transaction.Change != NilSleeve {
			l.wallet.coinStorage.Add(transaction.Change)
		}
		// Send receipt: Need ID of original invoice corresponding to this
		// transaction. That's something that the invoicing client should
		// be able to keep track of.
		l.wallet.completedOutboundPayments.Upsert(transaction.OriginID, transaction)
		receipt := l.formatReceipt(transaction)
		globals.N.DEBUG.Printf("Attempting to send receipt to transaction"+
			" recipient: %v!", transaction.Recipient)
		err := io.Messaging.SendMessage(transaction.Recipient,
			receipt.GetPayload())
		if err != nil {
			globals.N.ERROR.Printf("Payment response listener couldn't send"+
				" receipt: %v", err.Error())
		}
	}
	globals.N.DEBUG.Printf("Payment response: %v", response.Response)
}

func (l *ResponseListener) formatReceipt(transaction *Transaction) *parse.Message {
	return &parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_RECEIPT,
			Body: transaction.OriginID[:],
		},
		Sender:   l.wallet.session.GetCurrentUser().UserID,
		Receiver: transaction.Recipient,
		Nonce:    nil,
	}
}

type ReceiptListener struct {
	wallet *Wallet
}

func (rl *ReceiptListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	var invoiceID parse.MessageHash
	copy(invoiceID[:], msg.Body)
	transaction, ok := rl.wallet.outboundRequests.Pop(invoiceID)
	if !ok {
		globals.N.WARN.Printf("ReceiptListener: Heard an invalid receipt from %v"+
			": %q", msg.Sender, invoiceID)
	} else {
		// Mark the transaction in the log of completed transactions
		rl.wallet.completedInboundPayments.Upsert(invoiceID, transaction)
		// Add the user's new coins to coin storage
		rl.wallet.coinStorage.Add(transaction.Create)
		// Let the payment receipt UI listeners know that a payment's come in
		switchboard.Listeners.Speak(&parse.Message{
			TypedBody: parse.TypedBody{
				Type: parse.Type_PAYMENT_RECEIPT_UI,
				Body: invoiceID[:],
			},
			Sender:   msg.Sender,
			Receiver: 0,
			Nonce:    nil,
		})
	}
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

func (w *Wallet) GetCompletedOutboundPayment(id parse.MessageHash) (
	Transaction, bool) {
	transaction, ok := w.completedOutboundPayments.Get(id)
	if !ok {
		return Transaction{}, ok
	} else {
		return *transaction, ok
	}
}

func (w *Wallet) GetCompletedInboundPayment(id parse.MessageHash) (
	Transaction, bool) {
	transaction, ok := w.completedInboundPayments.Get(id)
	if !ok {
		return Transaction{}, ok
	} else {
		return *transaction, ok
	}
}

// TODO We could also switch on transaction list tags, but that would be slower.
// It's unclear to me which approach is better between the two.
type TransactionListID int

const (
	OutboundRequests          TransactionListID = iota
	InboundRequests
	PendingTransactions
	OutboundCompletedPayments
	InboundCompletedPayments
)

type TransactionListOrder int

const (
	TimestampDescending TransactionListOrder = iota
	TimestampAscending
	ValueDescending
	ValueAscending
)

// This structure is weird because it makes it easier to write an API that
// can go over gomobile
func (w *Wallet) GetTransactionIDs(id TransactionListID,
	order TransactionListOrder) []byte {
	switch id {
	case OutboundRequests:
		return w.outboundRequests.getKeys(order)
	case InboundRequests:
		return w.inboundRequests.getKeys(order)
	case PendingTransactions:
		return w.pendingTransactions.getKeys(order)
	case OutboundCompletedPayments:
		return w.completedOutboundPayments.getKeys(order)
	case InboundCompletedPayments:
		return w.completedInboundPayments.getKeys(order)
	default:
		return nil
	}
}
