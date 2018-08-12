package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"math"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Shows that CreateTransactionList creates new storage properly
func TestCreateTransactionList_New(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	// show that the ordered list does not exist
	key := "TestTransactionList"

	_, err := s.QueryMap(key)

	if err != user.ErrQuery {
		if err == nil {
			t.Errorf("CreateTransactionList: Transaction List returned when it should not exist")
		} else {
			t.Errorf("CreateTransactionList: Transaction List returned incorrect error when it should not exist: %s", err.Error())
		}
	}

	// create the ordered storage
	tl, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("CreateTransactionList: error returned on valid ordered storage creation: %s", err.Error())
	}

	if tl.session != s {
		t.Errorf("CreateTransactionList: does not point to session correctly")
	}

	if tl.value != 0 {
		t.Errorf("CreateTransactionList: inital value incorrect: Expected: %v, Recieved: %v", 0, tl.value)
	}

	if len(*tl.transactionMap) != 0 {
		t.Errorf("CreateTransactionList: new transactionList storage does not contain an empty list: %v", *tl.transactionMap)
	}

	_, err = s.QueryMap(key)

	if err != nil {
		t.Errorf("CreateTransactionList: Transaction Liste not created in storage")
	}
}

// Shows that CreateTransactionList loads old transaction List properly
func TestCreateTransactionList_Load(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	// show that the transaction list does not exist
	key := "TestTransactionList"

	tl, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("CreateTransactionList: error returned on valid ordered storage creation: %s", err.Error())
	}

	if tl.session != s {
		t.Errorf("CreateTransactionList: does not point to session correctly")
	}

	if tl.value != 0 {
		t.Errorf("CreateTransactionList: inital value incorrect: Expected: %v, Recieved: %v", 0, tl.value)
	}

	nt := Transaction{}

	(*tl.transactionMap)[parse.MessageHash{}] = &nt

	s.StoreSession()

	tl2, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("CreateTransactionList: error returned on valid transaction list creation: %s", err.Error())
	}

	if len(*tl2.transactionMap) != 1 {
		t.Errorf("CreateTransactionList: new transaction list does not contain data from old transaction list")
	}
}

// Shows

// Shows that TransactionList.Value returns the value of the storage correctly
func TestTransactionList_Value(t *testing.T) {

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	src := rand.NewSource(42)
	rng := rand.New(src)

	for i := 0; i < 100; i++ {
		value := rng.Uint64() % uint64(coin.MaxValueDenominationRegister)

		tl := &TransactionList{nil, value, s}

		if tl.Value() != value {
			t.Errorf("TransactionList.Value: Returned incorrect value: "+
				"Expected: %v, recieved: %v", value, tl.value)
		}
	}
}

// Shows that TransactionList.Upsert works when the list is empty
func TestTransactionList_Upsert_Empty(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	tMap := make(map[parse.MessageHash]*Transaction)

	tl := TransactionList{&tMap, 0, s}

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	tl.Upsert(t1Hash, &t1)

	if !reflect.DeepEqual(t1, *(*tl.transactionMap)[t1Hash]) {
		t.Errorf("TransactionList.Upsert: transaction not added to list properly; "+
			"Expected: %v, Recieved: %v", t1, *(*tl.transactionMap)[t1Hash])
	}
}

// Shows that TransactionList.Upsert works when the list is not empty
func TestTransactionList_Upsert_Multi(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	tMap := make(map[parse.MessageHash]*Transaction)
	tMap[t1Hash] = &t1

	tl := TransactionList{&tMap, 0, s}

	t2 := Transaction{Memo: "2"}
	t2Hash := parse.Message{TypedBody: parse.TypedBody{2, []byte{2}}}.Hash()

	tl.Upsert(t2Hash, &t2)

	if !reflect.DeepEqual(t2, *(*tl.transactionMap)[t2Hash]) {
		t.Errorf("TransactionList.Upsert: transaction not added to list properly; "+
			"Expected: %v, Recieved: %v", t2, *(*tl.transactionMap)[t2Hash])
	}
}

// Shows that TransactionList.Upsert's results are properly saved
func TestTransactionList_Upsert_Save(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestTransactionList"

	tl, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("TransactionList.Upsert: valid Transaction List not created properly: %s", err.Error())
	}

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	tl.Upsert(t1Hash, &t1)

	if !reflect.DeepEqual(t1, *(*tl.transactionMap)[t1Hash]) {
		t.Errorf("TransactionList.Upsert: transaction not added to list properly; "+
			"Expected: %v, Recieved: %v", t1, *(*tl.transactionMap)[t1Hash])
	}

	s.StoreSession()

	tl2, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("TransactionList.Upsert: valid Transaction List not loaded properly: %s", err.Error())
	}

	if !reflect.DeepEqual(t1, *(*tl2.transactionMap)[t1Hash]) {
		t.Errorf("TransactionList.Upsert: transaction not added to list properly; "+
			"Expected: %v, Recieved: %v", t1, *(*tl2.transactionMap)[t1Hash])
	}
}

// Shows that TransactionList.Get works when the list has multiple members
func TestTransactionList_Get(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	t2 := Transaction{Memo: "2"}
	t2Hash := parse.Message{TypedBody: parse.TypedBody{2, []byte{2}}}.Hash()

	tMap := make(map[parse.MessageHash]*Transaction)
	tMap[t1Hash] = &t1
	tMap[t2Hash] = &t2

	tl := TransactionList{&tMap, 0, s}

	//Test extant
	tGet, b := tl.Get(t2Hash)

	if !b {
		t.Errorf("TransactionList.Get: no transaction found with valid lookup hash")
	} else if !reflect.DeepEqual(t2, *tGet) {
		t.Errorf("TransactionList.Get: transaction returned that existes; "+
			"Expected: %v, Recieved: %v", t2, *tGet)
	}

	//Test non extant
	_, b = tl.Get(parse.MessageHash{})

	if b {
		t.Errorf("TransactionList.Get: transaction found with invalid lookup hash")
	}
}

// Shows that TransactionList.Pop works when the list is not empty
func TestTransactionList_Pop(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	t2 := Transaction{Memo: "2"}
	t2Hash := parse.Message{TypedBody: parse.TypedBody{2, []byte{2}}}.Hash()

	tMap := make(map[parse.MessageHash]*Transaction)
	tMap[t1Hash] = &t1
	tMap[t2Hash] = &t2

	tl := TransactionList{&tMap, 0, s}

	t1Pop, b := tl.Pop(t1Hash)

	if !b {
		t.Errorf("TransactionList.Pop: valid transaction not returned; ")
	}

	if !reflect.DeepEqual(t1, *t1Pop) {
		t.Errorf("TransactionList.Pop: transaction not returned properly; "+
			"Expected: %v, Recieved: %v", t1, *t1Pop)
	}

	_, bGet := tl.Get(t1Hash)

	if bGet {
		t.Errorf("TransactionList.Pop: transaction not deleted from List")
	}
}

// Shows that TransactionList.Pop errors properly when the element doesn't exist
func TestTransactionList_Pop_Invalid(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	t2 := Transaction{Memo: "2"}
	t2Hash := parse.Message{TypedBody: parse.TypedBody{2, []byte{2}}}.Hash()

	tMap := make(map[parse.MessageHash]*Transaction)
	tMap[t1Hash] = &t1
	tMap[t2Hash] = &t2

	tl := TransactionList{&tMap, 0, s}

	_, b := tl.Pop(parse.MessageHash{})

	if b {
		t.Errorf("TransactionList.Pop: error not recieved on invalid transaction; ")
	}
}

// Shows that TransactionList.Upsert works when the list is not empty
func TestTransactionList_Pop_Save(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestTransactionList"

	tl, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("TransactionList.Pop: valid Transaction List not created properly: %s", err.Error())
	}

	t1 := Transaction{Memo: "1"}
	t1Hash := parse.Message{TypedBody: parse.TypedBody{0, []byte{0}}}.Hash()

	t2 := Transaction{Memo: "2"}
	t2Hash := parse.Message{TypedBody: parse.TypedBody{2, []byte{2}}}.Hash()

	tl.Upsert(t1Hash, &t1)
	tl.Upsert(t2Hash, &t2)

	t1Pop, b := tl.Pop(t1Hash)

	if !b {
		t.Errorf("TransactionList.Pop: valid transaction not returned; ")
	}

	if !reflect.DeepEqual(t1, *t1Pop) {
		t.Errorf("TransactionList.Pop: transaction not returned properly; "+
			"Expected: %v, Recieved: %v", t1, *t1Pop)
	}

	_, bGet := tl.Get(t1Hash)

	if bGet {
		t.Errorf("TransactionList.Pop: transaction not deleted from List")
	}

	s.StoreSession()

	tl2, err := CreateTransactionList(key, s)

	if err != nil {
		t.Errorf("TransactionList.pOP: valid Transaction List not loaded properly: %s", err.Error())
	}

	_, b2Get := tl2.Get(t1Hash)

	if b2Get {
		t.Errorf("TransactionList.Pop: transaction not deleted from List after load")
	}

}

func TestTransactionList_GetKeysByTimestampDescending(t *testing.T) {
	// populate a transaction list with some items
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})
	transactionMap := make(map[parse.MessageHash]*Transaction)
	transactions := TransactionList{
		transactionMap: &transactionMap,
		session:        s,
	}

	keys := []string{
		"a,ir.g",
		"ri,a'p",
		"ouxr;a",
		"ai8,9p",
		"xrgdls",
	}

	times := []time.Time{
		time.Unix(1, 0),
		time.Unix(2, 0),
		time.Unix(3, 0),
		time.Unix(5, 0),
		time.Unix(4, 0),
	}

	ids := make([]parse.MessageHash, len(keys))
	for i := range ids {
		copy(ids[i][:], keys[i])
		transactions.Upsert(ids[i], &Transaction{
			Timestamp: times[i],
		})
	}

	// get the transactions sorted by their timestamp, most to least recent
	keyList := transactions.GetKeysByTimestampDescending()

	// verify that the keys we got correspond to correctly sorted transactions
	numKeys := uint64(len(keyList)) / parse.MessageHashLen
	var thisKey parse.MessageHash
	// Maxint Unix time is divided by two because golang's after function gets
	// weird with dates that are too far in the future
	lastTransactionTime := time.Unix(math.MaxInt64/2, 0)
	for i := uint64(0); i < numKeys; i++ {
		copy(thisKey[:], keyList[i*parse.MessageHashLen:])
		thisTransaction, ok := transactions.Get(thisKey)
		if !ok {
			t.Errorf("Couldn't get a transaction with the key %q", thisKey)
		} else {
			thisTransactionTime := thisTransaction.Timestamp
			// We should have the most recent transaction first
			if thisTransactionTime.After(lastTransactionTime) {
				t.Errorf("Transaction %v at time %v was after time %v", i,
					thisTransactionTime.Format(time.UnixDate),
					lastTransactionTime.Format(time.UnixDate))
			}
			lastTransactionTime = thisTransactionTime
		}
	}
}
