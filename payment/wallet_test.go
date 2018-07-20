package payment

import (
	"testing"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/client/parse"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/client/globals"
	"time"
	"reflect"
	"bytes"
)

// Tests whether invoice transactions get stored in the session correctly
func TestWallet_registerInvoice(t *testing.T) {
	payee := user.ID(1)
	payer := user.ID(2)
	memo := "for serious cryptography"
	value := uint64(85)

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{payee, "Taxman McGee"}, "",
		[]user.NodeKeys{})

	or, err := CreateTransactionList(OutboundRequestsTag, s)
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
	expected := Transaction{
		Create:    sleeve,
		Sender:    payer,
		Recipient: payee,
		Memo:      memo,
		Timestamp: time.Now(),
		Value:     value,
	}

	hash := parse.MessageHash{1, 2, 3, 4, 5}
	w.registerInvoice(hash, &expected)

	sessionReqs, err := s.QueryMap(OutboundRequestsTag)
	if err != nil {
		t.Error(err.Error())
	}

	actualReqs := sessionReqs.(map[parse.MessageHash]*Transaction)
	if len(actualReqs) != 1 {
		t.Error("Transaction map stored in outbound transactions contained" +
			" other transactions than expected")
	}
	actualReq := actualReqs[hash]
	if !reflect.DeepEqual(actualReq, &expected) {
		t.Error("Register invoice didn't match the invoice in the session")
	}
}

// Tests Invoice's message creation, and smoke tests the message's storage in
// the wallet's session
func TestWallet_Invoice(t *testing.T) {
	payee := user.ID(1)
	payer := user.ID(2)
	memo := "please gib"
	value := uint64(50)
	invoiceTime := time.Now()

	// Set up the wallet and its storage
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{payee, "Taxman McGee"}, "",
		[]user.NodeKeys{})

	or, err := CreateTransactionList(OutboundRequestsTag, s)
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
	if msg.Type != parse.Type_PAYMENT_INVOICE {
		t.Errorf("Invoice type didn't match. Got: %v, expected %v",
			msg.Type.String(), parse.Type_PAYMENT_INVOICE.String())
	}
	// Parse the body and make sure the fields are correct
	invoiceMsg := parse.PaymentInvoice{}
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

	actualReqs := sessionReqs.(map[parse.MessageHash]*Transaction)
	if len(actualReqs) != 1 {
		t.Error("Transaction map stored in outbound transactions contained" +
			" other transactions than expected")
	}
}

//Make sure the session stays untouched when passing malformed inputs to the
//invoice listener
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
			Type: parse.Type_NO_TYPE,
			Body: nil,
		},}, false)

	if s {
		t.Error("Invoice listener heard a message with the wrong type")
	}

	// Test 2: malformed proto buffer
	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_INVOICE,
			Body: []byte("fun fact: clownfish aren't actually very funny"),
		},
		Sender:   0,
		Receiver: 0,
		Nonce:    nil,
	}, false)

	if s {
		t.Error("Invoice listener heard a message with the right type, " +
			"but it wasn't actually an invoice")
	}

	// Test 3: good proto buffer, coin has wrong length
	invoice := parse.PaymentInvoice{
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
			Type: parse.Type_PAYMENT_INVOICE,
			Body: wireRep,
		},
	}, false)

	if s {
		t.Error("Invoice listener heard a message with short coin")
	}

	// Test 4: good proto buffer, coin has right length but wrong type
	erroneousCompound := [coin.BaseFrameLen]byte{0x04,}
	invoice.CreatedCoin = erroneousCompound[:]
	wireRep, err = proto.Marshal(&invoice)
	if err != nil {
		t.Error(err.Error())
	}

	invoiceListener.Hear(&parse.Message{
		TypedBody: parse.TypedBody{
			Type: parse.Type_PAYMENT_INVOICE,
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

func TestInvoiceListener_Hear(t *testing.T) {
	payee := user.ID(1)
	payer := user.ID(2)
	value := uint64(50)
	memo := "please gib"
	// Set up the wallet and its storage
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{payer, "CEO MF DOOM"}, "",
		[]user.NodeKeys{})

	ir, err := CreateTransactionList(InboundRequestsTag, s)
	if err != nil {
		t.Error(err.Error())
	}

	// Nil fields are there to make sure fields that shouldn't get touched
	// don't get touched
	w := Wallet{
		inboundRequests: ir,
		session:         s,
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
	var paymentInvoice parse.PaymentInvoice
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
	actualRequests := incomingRequests.(map[parse.MessageHash]*Transaction)
	if !reflect.DeepEqual(actualRequests[hash], req){
		t.Error("Request in incoming requests map didn't match received" +
			" request")
	}
}
