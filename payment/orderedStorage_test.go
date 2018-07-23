package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"math/rand"
	"reflect"
	"testing"
)

// Shows that CreateOrderedStorage creates new storage properly
func TestCreateOrderedStorage_New(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	// show that the ordered list does not exist
	key := "TestOrderedList"

	_, err := s.QueryMap(key)

	if err != user.ErrQuery {
		if err == nil {
			t.Errorf("CreateOrderedStorage: Ordered storage returned when it should not exist")
		} else {
			t.Errorf("CreateOrderedStorage: Ordered storage returned incorrect error when it should not exist: %s", err.Error())
		}
	}

	// create the ordered storage
	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("CreateOrderedStorage: error returned on valid ordered storage creation: %s", err.Error())
	}

	if ocs.session != s {
		t.Errorf("CreateOrderedStorage: does not point to session correctly")
	}

	if ocs.value != 0 {
		t.Errorf("CreateOrderedStorage: inital value incorrect: Expected: %v, Recieved: %v", 0, ocs.value)
	}

	if !reflect.DeepEqual([]coin.Sleeve{}, *ocs.list) {
		t.Errorf("CreateOrderedStorage: new ordered storage does not contain an empty list: %v", *ocs.list)
	}

	_, err = s.QueryMap(key)

	if err != nil {
		t.Errorf("CreateOrderedStorage: Ordered storage not created in storage")
	}
}

// Shows that CreateOrderedStorage loads old storage properly
func TestCreateOrderedStorage_Load(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	// show that the ordered list does not exist
	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("CreateOrderedStorage: error returned on valid ordered storage creation: %s", err.Error())
	}

	if ocs.session != s {
		t.Errorf("CreateOrderedStorage: does not point to session correctly")
	}

	if ocs.value != 0 {
		t.Errorf("CreateOrderedStorage: inital value incorrect: Expected: %v, Recieved: %v", 0, ocs.value)
	}

	ns, err := coin.NewSleeve(10)

	if err != nil {
		t.Errorf("CreateOrderedStorage: sleeve creation failed: %s", err.Error())
	}

	*ocs.list = append(*ocs.list, ns)

	s.StoreSession()

	ocs2, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("CreateOrderedStorage: error returned on valid ordered storage creation: %s", err.Error())
	}

	if !reflect.DeepEqual(ocs.list, ocs2.list) {
		t.Errorf("CreateOrderedStorage: new ordered storage does not contain data from old ordered storage: "+
			"old: %v, new: %v", *ocs.list, *ocs2.list)
	}
}

// Shows that OrderedCoinStorage.Value returns the value of the storage correctly
func TestOrderedCoinStorage_Value(t *testing.T) {

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	src := rand.NewSource(42)
	rng := rand.New(src)

	for i := 0; i < 100; i++ {
		value := rng.Uint64() % uint64(coin.MaxValueDenominationRegister)

		ocs := &OrderedCoinStorage{nil, value, s}

		if ocs.Value() != value {
			t.Errorf("OrderedCoinStorage.Value: Returned incorrect value: "+
				"Expected: %v, recieved: %v", value, ocs.value)
		}

	}
}

// Shows that OrderedCoinStorage.Add works when the list is empty
func TestOrderedCoinStorage_Add_Empty(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Add: sleeve creation failed: %s", err.Error())
	}

	ocs := OrderedCoinStorage{&[]coin.Sleeve{}, 0, s}

	ocs.Add(cs)

	if !reflect.DeepEqual(cs, (*ocs.list)[0]) {
		t.Errorf("OrderedCoinStorage.Add: coin sleeve not added to list: %s", err.Error())
	}
}

// Shows that OrderedCoinStorage.Add works when the list isn't empty and properly orders the list
func TestOrderedCoinStorage_Add_Multi(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	ocs := OrderedCoinStorage{&[]coin.Sleeve{}, 0, s}

	unorderdValues := []uint64{100, 13, 44}
	orderdValues := []uint64{13, 44, 100}

	for _, v := range unorderdValues {

		cs, err := coin.NewSleeve(v)

		if err != nil {
			t.Errorf("OrderedCoinStorage.Add: sleeve creation failed: %s", err.Error())
		}

		ocs.Add(cs)

	}

	if len(*ocs.list) != len(unorderdValues) {
		t.Errorf("OrderedCoinStorage.Add: List does not have the correct number of values after multipled adds: "+
			"Expected: %v, Recieved: %v, list: %v", len(unorderdValues), len(*ocs.list), *ocs.list)
	}

	valid := true

	for i := 0; i < len(orderdValues); i++ {
		if (*ocs.list)[i].Value() != orderdValues[i] {
			valid = false
		}
	}

	if !valid {
		t.Errorf("OrderedCoinStorage.Add: List not ordered ocrrectly after mutliple adds:"+
			"Expected: %v, Recieved: %v", orderdValues, *ocs.list)
	}
}

// Shows that added sleeves can be loaded after a save
func TestOrderedCoinStorage_Add_Save(t *testing.T) {

	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	// show that the ordered list does not exist
	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Add: error returned on valid ordered storage creation: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Add: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	s.StoreSession()

	ocs2, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Add: error returned on valid ordered storage creation: %s", err.Error())
	}

	if !reflect.DeepEqual(ocs.list, ocs2.list) {
		t.Errorf("OrderedCoinStorage.Add: new ordered storage does not contain data from old ordered storage: "+
			"old: %v, new: %v", *ocs.list, *ocs2.list)
	}
}

// Shows that a single sleeve can be popped from ordered coin storage
func TestOrderedCoinStorage_Pop_Single(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	csr := ocs.Pop(0)

	if !reflect.DeepEqual(cs, csr) {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not the same as created coin: "+
			"Created: %v, Popped: %v", cs, csr)
	}
}

// Shows that a can be popped from ordered coin storage
func TestOrderedCoinStorage_Pop_Single(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	csr := ocs.Pop(0)

	if !reflect.DeepEqual(cs, csr) {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not the same as created coin: "+
			"Created: %v, Popped: %v", cs, csr)
	}
}
