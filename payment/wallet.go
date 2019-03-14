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
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/coin"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
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

	// Listen to this switchboard to get UI messages from the wallet.
	// This includes the types PAYMENT_INVOICE_UI, PAYMENT_RESPONSE, and
	// PAYMENT_RECEIPT_UI.
	switchboard *switchboard.Switchboard
}

// Transaction lists are meant to be read-only when exported, so we return a
// copy of these pointers and only export non-mutating methods from the
// transaction list
func (w *Wallet) GetOutboundRequests() *TransactionList {
	return w.outboundRequests
}

func (w *Wallet) GetInboundRequests() *TransactionList {
	return w.inboundRequests
}

func (w *Wallet) GetPendingTransactions() *TransactionList {
	return w.pendingTransactions
}

func (w *Wallet) GetCompletedInboundPayments() *TransactionList {
	return w.completedInboundPayments
}

func (w *Wallet) GetCompletedOutboundPayments() *TransactionList {
	return w.completedOutboundPayments
}

func (w *Wallet) GetSwitchboard() *switchboard.Switchboard {
	return w.switchboard
}

// If you want the wallet to be able to receive messages you must register its
// listeners
// api.Login does this, while api.Register (during minting) does not
func CreateWallet(s user.Session, doMint bool) (*Wallet, error) {

	cs, err := CreateOrderedStorage(CoinStorageTag, s)

	if err != nil {
		return nil, err
	}

	if doMint {
		mintedCoins := coin.MintArbitrarily(s.GetCurrentUser().User[:])
		for i := range mintedCoins {
			cs.add(mintedCoins[i])
		}
	}

	obr, err := createTransactionList(OutboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	ibr, err := createTransactionList(InboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	pt, err := createTransactionList(PendingTransactionsTag, s)

	if err != nil {
		return nil, err
	}

	ip, err := createTransactionList(InboundPaymentsTag, s)

	if err != nil {
		return nil, err
	}

	op, err := createTransactionList(OutboundPaymentsTag, s)

	if err != nil {
		return nil, err
	}

	sb := switchboard.NewSwitchboard()

	w := &Wallet{
		coinStorage:               cs,
		outboundRequests:          obr,
		inboundRequests:           ibr,
		pendingTransactions:       pt,
		completedInboundPayments:  ip,
		completedOutboundPayments: op,
		session:                   s,
		switchboard:               sb,
	}

	return w, nil
}

// You need to call this method after creating the wallet to have the wallet
// behave correctly when receiving messages
// TODO: Should this take the listeners as parameters?
func (w *Wallet) RegisterListeners() {
	switchboard.Listeners.Register(id.ZeroID,
		format.None, int32(cmixproto.Type_PAYMENT_INVOICE),
		&InvoiceListener{
			wallet: w,
		})
	switchboard.Listeners.Register(getPaymentBotID(),
		format.None, int32(cmixproto.Type_PAYMENT_RESPONSE),
		&ResponseListener{
			wallet: w,
		})
	switchboard.Listeners.Register(id.ZeroID,
		format.None, int32(cmixproto.Type_PAYMENT_RECEIPT),
		&ReceiptListener{
			wallet: w,
		})
}

// Creates an invoice, which you can add to the wallet and create a message of
func createInvoice(payer *id.User, payee *id.User, value uint64,
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
	w.outboundRequests.upsert(invoice.OriginID, invoice)
	return w.session.StoreSession()
}

// Creates, formats, and registers an invoice in the outgoing requests
// Assumes that the payee is the current user in the session
func (w *Wallet) Invoice(payer *id.User, value int64,
	memo string) (*parse.Message, error) {

	if value <= 0 {
		return nil, errors.New("must request a non-zero, " +
			"positive amount of money for an invoice")
	}

	transaction, err := createInvoice(payer, w.session.GetCurrentUser().User,
		uint64(value), memo)
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

func (il *InvoiceListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	globals.Log.DEBUG.Printf("Heard an invoice from %v!", msg.GetSender())
	var invoice cmixproto.PaymentInvoice

	// Test for incorrect message type, just in case
	if msg.GetInnerType() != int32(cmixproto.Type_PAYMENT_INVOICE) {
		globals.Log.WARN.Printf("InvoiceListener: Got an invoice with the incorrect"+
			" type: %v", cmixproto.Type(msg.GetInnerType()).String())
		return
	}

	m := msg.(*parse.Message)

	// Don't humor people who send malformed messages
	if err := proto.Unmarshal(m.Body, &invoice); err != nil {
		globals.Log.WARN.Printf("InvoiceListener: Got error unmarshaling inbound"+
			" invoice: %v", err.Error())
		return
	}

	if uint64(len(invoice.CreatedCoin)) != coin.BaseFrameLen {
		globals.Log.WARN.Printf("InvoiceListener: Created coin has incorrect length"+
			" %v and is likely invalid", len(invoice.CreatedCoin))
		return
	}

	if !coin.IsCompound(invoice.CreatedCoin) {
		globals.Log.WARN.Printf("InvoiceListener: Got an invoice with an incorrect" +
			" coin type")
		return
	}

	// Convert the message to a compound
	var compound coin.Compound
	copy(compound[:], invoice.CreatedCoin)

	invoiceID := m.Hash()
	transaction := &Transaction{
		Create:    coin.ConstructSleeve(nil, &compound),
		Sender:    m.Receiver,
		Recipient: m.Sender,
		Memo:      invoice.Memo,
		Timestamp: time.Unix(invoice.Time, 0),
		Value:     compound.Value(),
		OriginID:  invoiceID,
	}

	// Actually add the request to the list of inbound requests
	il.wallet.inboundRequests.upsert(invoiceID, transaction)
	// and save it
	il.wallet.session.StoreSession()

	// The invoice UI message allows the UI to notify the user that the new
	// invoice is here and ready to be paid
	il.wallet.switchboard.Speak(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: int32(cmixproto.Type_PAYMENT_INVOICE_UI),
			Body: invoiceID[:],
		},
		Sender:   getPaymentBotID(),
		Receiver: id.ZeroID,
		Nonce:    nil,
	})
}

func getPaymentBotID() *id.User {
	return new(id.User).SetUints(&[4]uint64{0, 0, 0, 2})
}

func buildPaymentPayload(request, change coin.Sleeve,
	funds []coin.Sleeve) []byte {
	// The order of these doesn't matter because the coin's header determines
	// whether you are funding or destroying the coin on the payment bot.
	payload := make([]byte, 0, format.MP_PAYLOAD_LEN)
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
	transaction, ok := w.inboundRequests.pop(requestID)
	if !ok {
		return nil, errors.New("that request wasn't in the list of inbound" +
			" requests")
	}
	msg, err := w.pay(transaction)
	if err != nil {
		// Roll back the popping
		w.inboundRequests.upsert(requestID, transaction)
		return nil, err
	}
	errStore := w.session.StoreSession()
	if errStore != nil {
		// Roll back the popping
		w.inboundRequests.upsert(requestID, transaction)
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
	funds, change, err := w.coinStorage.Fund(inboundRequest.Value, 3)
	if err != nil {
		return nil, err
	}

	paymentMessage := buildPaymentPayload(inboundRequest.Create, change, funds)

	// FIXME This is only approximately correct, thanks to the padding
	if len(parse.TypeAsBytes(int32(cmixproto.Type_PAYMENT_TRANSACTION)))+
		len(paymentMessage) > format.MP_PAYLOAD_LEN {
		// The message is too long to fit in a single payment message
		panic("Payment message doesn't fit in a single message")
	}

	msg := parse.Message{
		TypedBody: parse.TypedBody{
			Type: int32(cmixproto.Type_PAYMENT_TRANSACTION),
			Body: paymentMessage,
		},
		Sender:   w.session.GetCurrentUser().User,
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
	globals.Log.INFO.Printf("Prepared payment message. Its ID is %v",
		base64.StdEncoding.EncodeToString(paymentID[:]))
	w.pendingTransactions.upsert(msg.Hash(), &pendingTransaction)

	// Return the result.
	return &msg, nil
}

type ResponseListener struct {
	wallet *Wallet
}

func (l *ResponseListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	var response cmixproto.PaymentResponse
	err := proto.Unmarshal(m.Body, &response)
	if err != nil {
		globals.Log.WARN.Printf("Heard an invalid response from the payment bot. "+
			"Error: %v", err.Error())
	}

	var paymentID parse.MessageHash
	copy(paymentID[:], response.ID)
	globals.Log.INFO.Printf("Heard response from payment bot. ID: %v",
		base64.StdEncoding.EncodeToString(paymentID[:]))
	transaction, ok := l.wallet.pendingTransactions.pop(paymentID)
	if !ok {
		globals.Log.ERROR.Printf("Couldn't find the transaction with that"+
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
		l.wallet.completedOutboundPayments.upsert(transaction.OriginID, transaction)
		receipt := l.formatReceipt(transaction)
		globals.Log.DEBUG.Printf("Attempting to send receipt to transaction"+
			" recipient: %v!", transaction.Recipient)
		err := io.Messaging.SendMessage(transaction.Recipient,
			receipt.Pack())
		if err != nil {
			globals.Log.ERROR.Printf("Payment response listener couldn't send"+
				" receipt: %v", err.Error())
		}
	}
	globals.Log.DEBUG.Printf("Payment response: %v", response.Response)
	l.wallet.switchboard.Speak(msg)
}

func (l *ResponseListener) formatReceipt(transaction *Transaction) *parse.Message {
	return &parse.Message{
		TypedBody: parse.TypedBody{
			Type: int32(cmixproto.Type_PAYMENT_RECEIPT),
			Body: transaction.OriginID[:],
		},
		Sender:   l.wallet.session.GetCurrentUser().User,
		Receiver: transaction.Recipient,
		Nonce:    nil,
	}
}

type ReceiptListener struct {
	wallet *Wallet
}

func (rl *ReceiptListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	var invoiceID parse.MessageHash
	copy(invoiceID[:], m.Body)
	transaction, ok := rl.wallet.outboundRequests.pop(invoiceID)
	if !ok {
		globals.Log.WARN.Printf("ReceiptListener: Heard an invalid receipt from %v"+
			": %q", m.Sender, invoiceID)
	} else {
		// Mark the transaction in the log of completed transactions
		rl.wallet.completedInboundPayments.upsert(invoiceID, transaction)
		// Add the user's new coins to coin storage
		rl.wallet.coinStorage.Add(transaction.Create)
		// Let the payment receipt UI listeners know that a payment's come in
		rl.wallet.switchboard.Speak(&parse.Message{
			TypedBody: parse.TypedBody{
				Type: int32(cmixproto.Type_PAYMENT_RECEIPT_UI),
				Body: invoiceID[:],
			},
			Sender:   m.Sender,
			Receiver: id.ZeroID,
			Nonce:    nil,
		})
	}
}

func (w *Wallet) GetAvailableFunds() uint64 {
	return w.coinStorage.Value()
}
