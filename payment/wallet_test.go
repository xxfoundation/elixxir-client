package payment

import (
	"testing"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/client/parse"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/client/globals"
	"bytes"
	"time"
	"reflect"
)

func TestWallet_Invoice(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{user.ID(1), "test"}, "", []user.NodeKeys{})

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

	// Request 50 unicoins from user 2
	memo := "please gib"
	value := uint64(50)
	invoiceTime := time.Now()
	msg, err := w.Invoice(user.ID(2), value, memo)
	if err != nil {
		t.Error(err.Error())
	}

	// Validate message
	if msg.Sender != s.GetCurrentUser().UserID {
		t.Errorf("Invoice sender didn't match. Got: %v, expected %v",
			msg.Sender, s.GetCurrentUser().UserID)
	}
	if msg.Receiver != user.ID(2) {
		t.Errorf("Invoice receiver didn't match. Got: %v, expected %v",
			msg.Receiver, user.ID(2))
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
	// make sure that we're within one second to avoid random,
	// infrequent failures
	if time.Now().Unix() - invoiceMsg.Time > 1 {
		t.Errorf("Invoice message time wasn't in acceptable bounds. Now: %v, " +
			"message time %v", invoiceTime.Unix(), invoiceMsg.Time)
	}
	if invoiceMsg.Memo != memo {
		t.Errorf("Invoice message memo didn't match input memo. Got: %v, " +
			"expected %v", invoiceMsg.Memo, memo)
	}
	// FIXME make sure nonce is populated

	// TODO This separate validation step indicates that this method is more
	// than one unit
	// Validate session object contents
	sessionReqs, err := s.QueryMap(OutboundRequestsTag)
	if err != nil {
		t.Error(err.Error())
	}
	// It doesn't seem to work quite how I'd hoped
	actualReqs := sessionReqs.(map[parse.MessageHash]*Transaction)
	if len(actualReqs) != 1 {
		t.Error("Transaction map stored in outbound transactions contained" +
			" other transactions than expected")
	}
	actualReq := actualReqs[msg.Hash()]
	expectedReq := Transaction{
		Create:    coin.ConstructSleeve(nil, compound),
		Sender:    s.GetCurrentUser().UserID,
		Recipient: user.ID(2),
		Memo:      memo,
		Timestamp: time.Unix(invoiceMsg.Time, 0),
		Value:     value,
	}
	if !reflect.DeepEqual(actualReq, &expectedReq) {
		t.Error("Transaction stored in map differed from expected")
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
