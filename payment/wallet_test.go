package payment

import (
	"testing"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/client/parse"
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/client/globals"
)

func TestWallet_Invoice(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{user.ID(1), "test"}, "", []user.NodeKeys{})

	or := TransactionList{
		transactionMap: make(map[parse.MessageHash]*Transaction),
		value:          0,
		session:        s,
	}

	w := Wallet{
		coinStorage:         nil,
		outboundRequests:    &or,
		inboundRequests:     nil,
		pendingTransactions: nil,
		session:             s,
	}

	// Request 50 unicoins from user 2
	msg, err := w.Invoice(user.ID(2), 50, "please gib")
	if err != nil {
		t.Error(err.Error())
	}

	if msg.Sender != s.GetCurrentUser().UserID {
		t.Errorf("Invoice sender didn't match. Got: %v, expected %v",
			msg.Sender, s.GetCurrentUser().UserID)
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
