package payment

import (
	"bytes"
	"fmt"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/crypto/cyclic"
	"reflect"
	"testing"
	"time"
	"gitlab.com/privategrity/client/cmixproto"
	"gitlab.com/privategrity/crypto/id"
)

// Tests whether invoice transactions get stored in the session correctly
func TestWallet_registerInvoice(t *testing.T) {
	payee := id.NewUserIDFromUint(1, t)
	payer := id.NewUserIDFromUint(2, t)
	memo := "for serious cryptography"
	value := uint64(85)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payee, Nick: "Taxman McGee"}, "",
		[]user.NodeKeys{})

	or, err := createTransactionList(OutboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	w := Wallet{
		coinStorage:         nil,
		outboundRequests:    or,
		inboundRequests:     nil,
		pendingTransactions: nil,
		session:             s,
	}

	sleeve, err := coin.NewSleeve(value)
	if err != nil {
		t.Error(err.Error())
	}
	hash := parse.MessageHash{1, 2, 3, 4, 5}
	expected := Transaction{
		Create:    sleeve,
		Sender:    payer,
		Recipient: payee,
		Memo:      memo,
		Timestamp: time.Now(),
		Value:     value,
		OriginID:  hash,
	}

	w.registerInvoice(&expected)

	sessionReqs, err := s.QueryMap(OutboundRequestsTag)
	if err != nil {
		t.Error(err.Error())
	}

	actualReqs := sessionReqs.(*map[parse.MessageHash]*Transaction)
	if len(*actualReqs) != 1 {
		t.Error("Transaction map stored in outbound transactions contained" +
			" other transactions than expected")
	}
	actualReq := (*actualReqs)[hash]
	if !reflect.DeepEqual(actualReq, &expected) {
		t.Error("Register invoice didn't match the invoice in the session")
	}
}

// Shows that CreateWallet creates new wallet properly
func TestCreateWallet(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: id.NewUserIDFromUint(1, t),
	Nick: "test"}, "", []user.NodeKeys{})

	_, err := CreateWallet(s, false)

	if err != nil {
		t.Errorf("CreateWallet: error returned on valid wallet creation: %s", err.Error())
	}

	//Test that Coin storage was added to the storage map properly
	_, err = s.QueryMap(CoinStorageTag)

	if err != nil {
		t.Errorf("CreateWallet: CoinStorage not created: %s", err.Error())
	}

	//Test that Outbound Request List was added to the storage map properly
	_, err = s.QueryMap(OutboundRequestsTag)

	if err != nil {
		t.Errorf("CreateWallet: Outbound Request List not created: %s", err.Error())
	}

	//Test that Inbound Request was added to the storage map properly
	_, err = s.QueryMap(InboundRequestsTag)

	if err != nil {
		t.Errorf("CreateWallet: Inbound Request List not created: %s", err.Error())
	}

	//Test that Pending Transaction List Request was added to the storage map properly
	_, err = s.QueryMap(PendingTransactionsTag)

	if err != nil {
		t.Errorf("CreateWallet: Pending Transaction List not created: %s", err.Error())
	}

}

// Tests Invoice's message creation, and smoke tests the message's storage in
// the wallet's session
func TestWallet_Invoice(t *testing.T) {
	payee := id.NewUserIDFromUint(1, t)
	payer := id.NewUserIDFromUint(2, t)
	memo := "please gib"
	value := int64(50)
	invoiceTime := time.Now()

	// Set up the wallet and its storage
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payee, Nick: "Taxman McGee"}, "",
		[]user.NodeKeys{})

	or, err := createTransactionList(OutboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Nil fields are there to make sure fields that shouldn't get touched
	// don't get touched
	w := Wallet{
		outboundRequests: or,
		session:          s,
	}

	msg, err := w.Invoice(payer, value, memo)
	if err != nil {
		t.Error(err.Error())
	}

	//Validate message
	if msg.Sender != payee {
		t.Errorf("Invoice sender didn't match. Got: %v, expected %v",
			msg.Sender, payee)
	}
	if msg.Receiver != payer {
		t.Errorf("Invoice receiver didn't match. Got: %v, expected %v",
			msg.Receiver, payer)
	}
	if msg.Type != cmixproto.Type_PAYMENT_INVOICE {
		t.Errorf("Invoice type didn't match. Got: %v, expected %v",
			msg.Type.String(), cmixproto.Type_PAYMENT_INVOICE.String())
	}
	// Parse the body and make sure the fields are correct
	invoiceMsg := cmixproto.PaymentInvoice{}
	err = proto.Unmarshal(msg.Body, &invoiceMsg)
	if err != nil {
		t.Error(err.Error())
	}
	request, ok := or.Get(msg.Hash())
	if !ok {
		t.Error("Couldn't get outbound request out of the wallet's list")
	}
	compound := request.Create.Compound()
	t.Logf("Added compound: %q", compound)
	if !bytes.Equal(invoiceMsg.CreatedCoin, compound[:]) {
		t.Error("Created coin in invoice message and outbound request's" +
			" compound differed")
	}
	//make sure that we're within one second to avoid random,
	//infrequent failures
	if time.Now().Unix()-invoiceMsg.Time > 1 {
		t.Errorf("Invoice message time wasn't in acceptable bounds. Now: %v, "+
			"message time %v", invoiceTime.Unix(), invoiceMsg.Time)
	}
	if invoiceMsg.Memo != memo {
		t.Errorf("Invoice message memo didn't match input memo. Got: %v, "+
			"expected %v", invoiceMsg.Memo, memo)
	}
	//FIXME make sure nonce is populated

	// Make sure there's exactly one entry in the session
	sessionReqs, err := s.QueryMap(OutboundRequestsTag)
	if err != nil {
		t.Error(err.Error())
	}

	actualReqs := sessionReqs.(*map[parse.MessageHash]*Transaction)
	if len(*actualReqs) != 1 {
		t.Error("Transaction map stored in outbound transactions contained" +
			" other transactions than expected")
	}
}

// Make sure the session stays untouched when passing malformed inputs to the
// invoice listener
func TestInvoiceListener_Hear_Errors(t *testing.T) {
	var s MockSession
	w := Wallet{
		session: &s,
	}

	invoiceListener := InvoiceListener{
		wallet: &w,
	}

	// Test 1: incorrect message type
	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_NO_TYPE,
			Body: nil,
		}}, false)

	if s {
		t.Error("Invoice listener heard a message with the wrong type")
	}

	// Test 2: malformed proto buffer
	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_INVOICE,
			Body: []byte("fun fact: clownfish aren't actually very funny"),
		},
		Sender:   id.ZeroID,
		Receiver: id.ZeroID,
		Nonce:    nil,
	}, false)

	if s {
		t.Error("Invoice listener heard a message with the right type, " +
			"but it wasn't actually an invoice")
	}

	// Test 3: good proto buffer, coin has wrong length
	invoice := cmixproto.PaymentInvoice{
		Time:        0,
		CreatedCoin: []byte{0xaa},
		Memo:        "",
	}
	wireRep, err := proto.Marshal(&invoice)
	if err != nil {
		t.Error(err.Error())
	}

	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_INVOICE,
			Body: wireRep,
		},
	}, false)

	if s {
		t.Error("Invoice listener heard a message with short coin")
	}

	// Test 4: good proto buffer, coin has right length but wrong type
	erroneousCompound := [coin.BaseFrameLen]byte{0x04}
	invoice.CreatedCoin = erroneousCompound[:]
	wireRep, err = proto.Marshal(&invoice)
	if err != nil {
		t.Error(err.Error())
	}

	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_INVOICE,
			Body: wireRep,
		},
	}, false)

	if s {
		t.Error("Invoice listener heard a message with a coin of wrong type")
	}
}

// the mock session is lava and if you touch it it's instant death
type MockSession bool

func (ms *MockSession) GetCurrentUser() (currentUser *user.User) {
	*ms = true
	return nil
}

func (ms *MockSession) GetGWAddress() string {
	*ms = true
	return ""
}

func (ms *MockSession) SetGWAddress(addr string) {
	*ms = true
}

func (ms *MockSession) GetKeys() []user.NodeKeys {
	*ms = true
	return nil
}

func (ms *MockSession) GetPrivateKey() *cyclic.Int {
	*ms = true
	return nil
}

func (ms *MockSession) GetPublicKey() *cyclic.Int {
	*ms = true
	return nil
}

func (ms *MockSession) StoreSession() error {
	*ms = true
	return nil
}

func (ms *MockSession) Immolate() error {
	*ms = true
	return nil
}

func (ms *MockSession) UpsertMap(key string, element interface{}) error {
	*ms = true
	return nil
}

func (ms *MockSession) QueryMap(key string) (interface{}, error) {
	*ms = true
	return nil, nil
}

func (ms *MockSession) DeleteMap(key string) error {
	*ms = true
	return nil
}

func (ms *MockSession) LockStorage() {
	*ms = true
}

func (ms *MockSession) UnlockStorage() {
	*ms = true
}
func (ms *MockSession) GetLastMessageID() string {
	*ms = true
	return ""
}

func (ms *MockSession) SetLastMessageID(id string) {
	*ms = true
}

func TestInvoiceListener_Hear(t *testing.T) {
	payee := id.NewUserIDFromUint(1, t)
	payer := id.NewUserIDFromUint(2, t)
	value := uint64(50)
	memo := "please gib"
	// Set up the wallet and its storage
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "CEO MF DOOM"}, "",
		[]user.NodeKeys{})

	ir, err := createTransactionList(InboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Nil fields are there to make sure fields that shouldn't get touched
	// don't get touched
	w := Wallet{
		inboundRequests: ir,
		session:         s,
		switchboard:     switchboard.NewSwitchboard(),
	}

	invoiceListener := InvoiceListener{wallet: &w}

	invoiceTransaction, err := createInvoice(payer, payee, value, memo)
	msg := invoiceTransaction.FormatPaymentInvoice()
	hash := msg.Hash()

	invoiceListener.Hear(msg, false)

	req, ok := ir.Get(hash)
	if !ok {
		t.Error("Couldn't get invoice message from inbound requests structure")
	}
	// Memo, payer, payee, value should all be equal
	if req.Memo != memo {
		t.Errorf("Memo didn't match. Got: %v, expected %v", req.Memo, memo)
	}
	if req.Sender != payer {
		t.Errorf("Payer didn't match. Got: %v, expected %v", req.Sender, payer)
	}
	if req.Recipient != payee {
		t.Errorf("Payee didn't match. Got: %v, expected %v", req.Recipient,
			payee)
	}
	if req.Value != value {
		t.Errorf("Value didn't match. Got: %v, expected %v", req.Value, value)
	}
	// The coin itself is a special case. You shouldn't send a coin's seed
	// over the network, except to the payment bot when you're proving that you
	// own a coin
	// So, the resulting coin's seed should be nil and its compound should be
	// the same
	if req.Create.Seed() != nil {
		t.Error("Created coin's seed wasn't nil on the message's recipient")
	}
	if *req.Create.Compound() != *invoiceTransaction.Create.Compound() {
		t.Error("Created coin's compounds weren't equal")
	}

	// Need to unmarshal the message to get the expected timestamp out of it
	var paymentInvoice cmixproto.PaymentInvoice
	proto.Unmarshal(msg.Body, &paymentInvoice)
	// The timestamp on the network message is only precise up to a second,
	// so we have to compare the timestamp on the outgoing network message
	// to the timestamp on the incoming network message rather than to the
	// timestamp on the transaction.
	if req.Timestamp.Unix() != paymentInvoice.Time {
		t.Errorf("Timestamp differed from expected. Got %v, expected %v",
			req.Timestamp.Unix(), paymentInvoice.Time)
	}

	// Now, verify that the session contains the same request
	incomingRequests, err := s.QueryMap(InboundRequestsTag)
	if err != nil {
		t.Error(err.Error())
	}
	actualRequests := incomingRequests.(*map[parse.MessageHash]*Transaction)
	if !reflect.DeepEqual((*actualRequests)[hash], req) {
		t.Error("Request in incoming requests map didn't match received" +
			" request")
	}
}

func TestWallet_Invoice_Error(t *testing.T) {
	payee := id.NewUserIDFromUint(1, t)
	payer := id.NewUserIDFromUint(2, t)
	memo := "please gib"
	// A value of zero should cause an error
	value := int64(0)

	// Set up the wallet and its storage
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payee, Nick: "Taxman McGee"}, "",
		[]user.NodeKeys{})

	or, err := createTransactionList(OutboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Nil fields are there to make sure fields that shouldn't get touched
	// don't get touched
	w := Wallet{
		outboundRequests: or,
		session:          s,
	}

	_, err = w.Invoice(payer, value, memo)
	if err == nil {
		t.Error("Didn't get an error for a worthless invoice")
	}

	// A value greater than the greatest possible value should cause an error
	value = int64(coin.MaxValueDenominationRegister) + 1

	_, err = w.Invoice(payer, value, memo)
	if err == nil {
		t.Error("Didn't get an error for an invoice that's too large")
	}
}

type MockMessaging struct{}

func (m *MockMessaging) SendMessage(recipientID *id.UserID,
	message []byte) error {
	return nil
}

func (m *MockMessaging) MessageReceiver(delay time.Duration) {}

func TestResponseListener_Hear(t *testing.T) {
	payer := id.NewUserIDFromUint(5, t)
	payee := id.NewUserIDFromUint(12, t)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "Darth Icky"}, "",
		[]user.NodeKeys{})

	walletAmount := uint64(8970)
	paymentAmount := uint64(962)
	changeAmount := walletAmount - paymentAmount

	storage, err := CreateOrderedStorage(CoinStorageTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	walletSleeve, err := coin.NewSleeve(walletAmount)
	if err != nil {
		t.Error(err.Error())
	}

	// We don't add the coin to the storage to concisely emulate the Pay()
	// method, which would remove the coin that we had added, and which should
	// always get called before getting a response from the payment bot.

	changeSleeve, err := coin.NewSleeve(changeAmount)
	if err != nil {
		t.Error(err.Error())
	}

	paymentSleeve, err := coin.NewSleeve(paymentAmount)
	if err != nil {
		t.Error(err.Error())
	}

	pt, err := createTransactionList(PendingTransactionsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// the contents of the sleeve don't actually matter as long as we say the
	// transaction succeeded. so we can just use the same sleeve for creation
	// and destruction

	// create the pending wallet transaction
	transaction := Transaction{
		Create:    paymentSleeve,
		Destroy:   []coin.Sleeve{walletSleeve},
		Change:    changeSleeve,
		Sender:    payer,
		Recipient: payee,
		Memo:      "for midichlorians and midichlorian paraphernalia",
		Timestamp: time.Now(),
		Value:     paymentAmount,
	}
	// for the purposes of this test the hash could be anything,
	// as long it's the same for the key to the map and in the return message
	var hash parse.MessageHash
	copy(hash[:], []byte("even though this hash may seem unlikely to the"+
		" casual observer, it is in fact a valid, real, and correct message hash"))
	pt.upsert(hash, &transaction)

	op, err := createTransactionList(OutboundPaymentsTag, s)

	// Create wallet that has the compound coins in it to do a payment
	// Unaffected lists are unpopulated
	w := Wallet{
		coinStorage:               storage,
		pendingTransactions:       pt,
		completedOutboundPayments: op,
		session:                   s,
		switchboard:               switchboard.NewSwitchboard(),
	}

	response := cmixproto.PaymentResponse{
		Success:  true,
		Response: "200 OK",
		ID:       hash[:],
	}
	// marshal response into a parse message
	wire, err := proto.Marshal(&response)

	listener := ResponseListener{wallet: &w}

	// The payment response listener sends a receipt to the invoice originator.
	// To prevent actually hitting the network in this test, replace the
	// messaging with a mock that doesn't send anything
	io.Messaging = &MockMessaging{}

	listener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_RESPONSE,
			Body: wire,
		},
		Sender:   payer,
		Receiver: payee,
		Nonce:    nil,
	}, false)

	// In the success case, the transaction is no longer pending because it
	// succeeded.
	if len(*w.pendingTransactions.transactionMap) != 0 {
		t.Error("There should be zero transactions pending in the map" +
			" after receiving a successful payment response.")
	}
	if w.pendingTransactions.Value() != 0 {
		t.Errorf("Pending transactions' total value should be zero after"+
			" receiving the payment response. It was %v",
			w.pendingTransactions.Value())
	}
	// After a successful transaction,
	// the coin storage should have the change in it
	if w.coinStorage.Value() != changeAmount {
		t.Errorf("Wallet didn't have value equal to the value of the change. "+
			"Got %v, expected %v", w.coinStorage.Value(), changeAmount)
	}

	// After a successful transaction, we should have the transaction's value
	// recorded in the outbound payments list for posterity.
	if w.completedOutboundPayments.Value() != paymentAmount {
		t.Errorf("Outbound payments didn't have the value expected. Got: %v, "+
			"expected %v", w.completedOutboundPayments.Value(), paymentAmount)
	}
}

func TestResponseListener_Hear_Failure(t *testing.T) {
	payer := id.NewUserIDFromUint(5, t)
	payee := id.NewUserIDFromUint(12, t)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "Darth Icky"}, "",
		[]user.NodeKeys{})

	walletAmount := uint64(8970)
	paymentAmount := uint64(962)
	changeAmount := walletAmount - paymentAmount

	storage, err := CreateOrderedStorage(CoinStorageTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	walletSleeve, err := coin.NewSleeve(walletAmount)
	if err != nil {
		t.Error(err.Error())
	}

	// We don't add the coin to the storage to concisely emulate the Pay()
	// method, which would remove the coin that we had added, and which should
	// always get called before getting a response from the payment bot.

	changeSleeve, err := coin.NewSleeve(changeAmount)
	if err != nil {
		t.Error(err.Error())
	}

	paymentSleeve, err := coin.NewSleeve(paymentAmount)
	if err != nil {
		t.Error(err.Error())
	}

	pt, err := createTransactionList(PendingTransactionsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// the contents of the sleeve don't actually matter as long as we say the
	// transaction succeeded. so we can just use the same sleeve for creation
	// and destruction

	// create the pending wallet transaction
	transaction := Transaction{
		Create:    paymentSleeve,
		Destroy:   []coin.Sleeve{walletSleeve},
		Change:    changeSleeve,
		Sender:    payer,
		Recipient: payee,
		Memo:      "for midichlorians and midichlorian paraphernalia",
		Timestamp: time.Now(),
		Value:     paymentAmount,
	}
	// for the purposes of this test the hash could be anything,
	// as long it's the same for the key to the map and in the return message
	var hash parse.MessageHash
	copy(hash[:], []byte("even though this hash may seem unlikely to the"+
		" casual observer, it is in fact a valid, real, and correct message hash"))
	pt.upsert(hash, &transaction)

	// Create wallet that has the compound coins in it to do a payment
	// Unaffected lists are unpopulated
	w := Wallet{
		coinStorage:         storage,
		pendingTransactions: pt,
		session:             s,
		switchboard:         switchboard.NewSwitchboard(),
	}

	response := cmixproto.PaymentResponse{
		Success: false,
		// The payment bot doesn't actually respond with a 404.
		// Also, if you've read this far, you have my deepest admiration.
		Response: "404 Not Found",
		ID:       hash[:],
	}
	// marshal response into a parse message
	wire, err := proto.Marshal(&response)

	listener := ResponseListener{wallet: &w}
	listener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_RESPONSE,
			Body: wire,
		},
		Sender:   payer,
		Receiver: payee,
		Nonce:    nil,
	}, false)

	// In the failure case, the transaction is no longer pending because it's
	// declined. The payee must invoice again if they want to retry the payment.
	if len(*w.pendingTransactions.transactionMap) != 0 {
		t.Error("There should be zero transactions pending in the map" +
			" after receiving a payment response.")
	}
	if w.pendingTransactions.Value() != 0 {
		t.Errorf("Pending transactions' total value should be zero after"+
			" receiving the payment response. It was %v",
			w.pendingTransactions.Value())
	}
	// The wallet should be restored to its original value after the
	// failed transaction
	if w.coinStorage.Value() != walletAmount {
		t.Errorf("Wallet didn't have value equal to the value of the change. "+
			"Got %v, expected %v", w.coinStorage.Value(), walletAmount)
	}
}

func TestWallet_Pay_NoChange(t *testing.T) {
	payer := id.NewUserIDFromUint(5, t)
	payee := id.NewUserIDFromUint(12, t)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "Darth Icky"}, "",
		[]user.NodeKeys{})

	paymentAmount := uint64(5008)
	walletAmount := uint64(5008)

	storage, err := CreateOrderedStorage(CoinStorageTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	sleeve, err := coin.NewSleeve(walletAmount)
	if err != nil {
		t.Error(err.Error())
	}
	storage.Add(sleeve)

	pt, err := createTransactionList(PendingTransactionsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Create wallet that has the compound coins in it to do a payment
	// Unaffected lists are unpopulated
	w := Wallet{
		coinStorage:         storage,
		pendingTransactions: pt,
		session:             s,
	}

	inboundTransaction, err := createInvoice(payer, payee, paymentAmount,
		"podracer maintenance")
	if err != nil {
		t.Error(err.Error())
	}

	msg, err := w.pay(inboundTransaction)
	if err != nil {
		t.Error(err.Error())
	}

	// verify message contents
	if !bytes.Contains(msg.Body, sleeve.Seed()[:]) {
		t.Error("Message body didn't contain the payment's source")
	}
	if !bytes.Contains(msg.Body, inboundTransaction.Create.Compound()[:]) {
		t.Error("Message body didn't contain the payment's destination")
	}
	if !(uint64(len(msg.Body)) == 2*coin.BaseFrameLen) {
		t.Error("Message body should have exactly two coins in it, " +
			"but it doesn't")
	}

	// verify wallet contents
	transaction, ok := w.pendingTransactions.Get(msg.Hash())
	if !ok {
		t.Error("Couldn't find the transaction in the map")
	} else {
		if transaction.Create != inboundTransaction.Create {
			t.Error("The transactions are creating different coins")
		}
		if transaction.Value != inboundTransaction.Value {
			t.Error("The transactions have different values")
		}
		if transaction.Sender != inboundTransaction.Sender {
			t.Error("The transactions have a different sender")
		}
		if transaction.Recipient != inboundTransaction.Recipient {
			t.Error("The transactions have a different recipient")
		}
		if transaction.Timestamp != inboundTransaction.Timestamp {
			t.Error("The transactions have a different timestamp")
		}
		if transaction.Memo != inboundTransaction.Memo {
			t.Error("The transactions have a different memo")
		}
		if transaction.Change != NilSleeve {
			t.Error("There shouldn't have been change for this transaction")
		}
		if !reflect.DeepEqual(transaction.Destroy[0], sleeve) {
			t.Error("The destroyed coin and the coin we forged to test the" +
				" transaction weren't identical")
		}
	}

	// TODO verify session contents? or do the wallet tests cover that enough?
}

func TestWallet_Pay_YesChange(t *testing.T) {
	payer := id.NewUserIDFromUint(5, t)
	payee := id.NewUserIDFromUint(12, t)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "Darth Icky"}, "",
		[]user.NodeKeys{})

	paymentAmount := uint64(2611)
	walletAmount := uint64(5008)

	storage, err := CreateOrderedStorage(CoinStorageTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	sleeve, err := coin.NewSleeve(walletAmount)
	if err != nil {
		t.Error(err.Error())
	}
	storage.Add(sleeve)

	pt, err := createTransactionList(PendingTransactionsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Create wallet that has the compound coins in it to do a payment
	// Unaffected lists are unpopulated
	w := Wallet{
		coinStorage:         storage,
		pendingTransactions: pt,
		session:             s,
	}

	inboundTransaction, err := createInvoice(payer, payee, paymentAmount,
		"podracer maintenance")
	if err != nil {
		t.Error(err.Error())
	}

	msg, err := w.pay(inboundTransaction)
	if err != nil {
		t.Error(err.Error())
	}

	// verify message contents
	if !bytes.Contains(msg.Body, sleeve.Seed()[:]) {
		t.Error("Message body didn't contain the payment's source")
	}
	if !bytes.Contains(msg.Body, inboundTransaction.Create.Compound()[:]) {
		t.Error("Message body didn't contain the payment's destination")
	}
	// look for the change and make sure it's reasonable
	for i := uint64(0); i < uint64(len(msg.Body)); i += coin.BaseFrameLen {
		thisCoin := msg.Body[i : i+coin.BaseFrameLen]
		if coin.IsCompound(msg.Body[i : i+coin.BaseFrameLen]) {
			if !bytes.Equal(inboundTransaction.Create.Compound()[:], thisCoin) {
				// we've found the change
				// make a compound with it and see if the value is correct
				var compound coin.Compound
				copy(compound[:], thisCoin)
				if compound.Value() != walletAmount-paymentAmount {
					t.Error("Change in the message didn't have the right value")
				}
			}
		}
	}

	if !(uint64(len(msg.Body)) == 3*coin.BaseFrameLen) {
		t.Error("Message body should have exactly three coins in it, " +
			"but it doesn't")
	}

	// verify wallet contents
	transaction, ok := w.pendingTransactions.Get(msg.Hash())
	if !ok {
		t.Error("Couldn't find the transaction in the map")
	} else {
		if transaction.Create != inboundTransaction.Create {
			t.Error("The transactions are creating different coins")
		}
		if transaction.Value != inboundTransaction.Value {
			t.Error("The transactions have different values")
		}
		if transaction.Sender != inboundTransaction.Sender {
			t.Error("The transactions have a different sender")
		}
		if transaction.Recipient != inboundTransaction.Recipient {
			t.Error("The transactions have a different recipient")
		}
		if transaction.Timestamp != inboundTransaction.Timestamp {
			t.Error("The transactions have a different timestamp")
		}
		if transaction.Memo != inboundTransaction.Memo {
			t.Error("The transactions have a different memo")
		}
		if transaction.Change.Value() != walletAmount-paymentAmount {
			t.Error("Incorrect amount of change for this transaction")
		}
		if !reflect.DeepEqual(transaction.Destroy[0], sleeve) {
			t.Error("The destroyed coin and the coin we forged to test the" +
				" transaction weren't identical")
		}
	}

	// TODO verify session contents
}

type ReceiptUIListener struct {
	hasHeard       bool
	gotTransaction bool
	w              *Wallet
}

func (rl *ReceiptUIListener) Hear(msg *parse.Message, isHeardElsewhere bool) {
	rl.hasHeard = true
	var invoiceID parse.MessageHash
	copy(invoiceID[:], msg.Body)
	_, rl.gotTransaction = rl.w.GetCompletedInboundPayments().Get(invoiceID)
	fmt.Printf("Heard receipt in the UI. Receipt sender: %q, invoice id %q\n",
		*msg.Sender, msg.Body)
}

// Tests the side effects of getting a receipt for a transaction that you
// sent out an invoice for
func TestReceiptListener_Hear(t *testing.T) {
	payer := id.NewUserIDFromUint(5, t)
	payee := id.NewUserIDFromUint(12, t)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{UserID: payer, Nick: "Darth Icky"}, "",
		[]user.NodeKeys{})

	walletAmount := uint64(8970)
	paymentAmount := uint64(1234)

	storage, err := CreateOrderedStorage(CoinStorageTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	walletSleeve, err := coin.NewSleeve(walletAmount)
	if err != nil {
		t.Error(err.Error())
	}
	storage.add(walletSleeve)

	or, err := createTransactionList(OutboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	var invoiceID parse.MessageHash
	copy(invoiceID[:], "you can make haute cuisine with dog biscuits")
	invoice, err := createInvoice(payer, payee, paymentAmount, "for counting to four")
	if err != nil {
		t.Error(err.Error())
	}
	or.upsert(invoiceID, invoice)

	ip, err := createTransactionList(InboundPaymentsTag, s)
	if err != nil {
		t.Error(err.Error())
	}
	w := &Wallet{
		coinStorage:              storage,
		outboundRequests:         or,
		completedInboundPayments: ip,
		session:                  s,
		switchboard:              switchboard.NewSwitchboard(),
	}

	listener := ReceiptListener{
		wallet: w,
	}

	// Test the register UI listener as well
	uiListener := &ReceiptUIListener{
		w: w,
	}
	w.switchboard.Register(id.ZeroID, cmixproto.Type_PAYMENT_RECEIPT_UI,
		uiListener)

	listener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: cmixproto.Type_PAYMENT_RECEIPT,
			Body: invoiceID[:],
		},
		Sender:   invoice.Sender,
		Receiver: invoice.Recipient,
		Nonce:    nil,
	}, false)

	if err != nil {
		t.Error(err.Error())
	}

	// make sure the UI gets informed afterwards
	if !uiListener.hasHeard {
		t.Error("UI listener hasn't heard the UI message")
	}

	// make sure the UI can get the transaction from the correct list
	if !uiListener.gotTransaction {
		t.Error("UI listener couldn't get the transaction from the list of" +
			" completed payments")
	}

	// Ensure correct state of wallet transaction lists after hearing receipt
	if w.outboundRequests.Value() != 0 {
		t.Errorf("Wallet outboundrequests value should be zero. Got: %v",
			w.outboundRequests.Value())
	}
	if w.completedInboundPayments.Value() != paymentAmount {
		t.Errorf("Wallet inboundpayments value should be the value of the"+
			" payment. Got %v, expected %v.", w.completedInboundPayments.Value(), paymentAmount)
	}
	if w.coinStorage.Value() != paymentAmount+walletAmount {
		t.Errorf("Expected funds to be added to the wallet upon receipt. "+
			"Got total value %v, expected %v.", w.coinStorage.Value(),
			paymentAmount+walletAmount)
	}
}
