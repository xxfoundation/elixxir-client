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

// Shows that a single sleeve can be gotten from ordered coin storage
func TestOrderedCoinStorage_Get(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Get: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Get: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	csr, b := ocs.Get(0)

	if !b {
		t.Errorf("OrderedCoinStorage.Get: could not find valid coin")
	}

	if !reflect.DeepEqual(cs, csr) {
		t.Errorf("OrderedCoinStorage.Get: gotten coin not the same as created coin: "+
			"Created: %v, Got: %v", cs, csr)
	}

	if len(*ocs.list) != 1 {
		t.Errorf("OrderedCoinStorage.Get: gotten coin not present in list")
	}
}

// Shows that a single sleeve can be popped from ordered coin storage
func TestOrderedCoinStorage_Pop(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	csr, b := ocs.Pop(0)

	if !b {
		t.Errorf("OrderedCoinStorage.Pop: could not find valid coin")
	}

	if !reflect.DeepEqual(cs, csr) {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not the same as created coin: "+
			"Created: %v, Popped: %v", cs, csr)
	}

	if len(*ocs.list) != 0 {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not removed from list")
	}
}

// Shows that when a sleeve is popped from ordered coin storage the result can be saved properly
func TestOrderedCoinStorage_Pop_Save(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	if len(*ocs.list) == 0 {
		t.Errorf("OrderedCoinStorage.Pop: added coin not present in list")
	}

	s.StoreSession()

	ocs2, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Pop: ordered storage creation failed: %s", err.Error())
	}

	csr, b := ocs2.Pop(0)

	if !b {
		t.Errorf("OrderedCoinStorage.Pop: could not find valid coin")
	}

	if !reflect.DeepEqual(cs, csr) {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not the same as created coin: "+
			"Created: %v, Popped: %v", cs, csr)
	}

	if len(*ocs2.list) != 0 {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not removed from list")
	}

	ocs3, err := CreateOrderedStorage(key, s)

	if len(*ocs3.list) != 0 {
		t.Errorf("OrderedCoinStorage.Pop: popped coin not removed from saved list")
	}
}

// Test that fund responds correct with insufficient funds
func TestOrderedCoinStorage_Fund_Insufficient(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	dst, chng, err := ocs.Fund(100, 1)

	if err != ErrInsufficientFunds {
		if err == nil {
			t.Errorf("OrderedCoinStorage.Fund: Did not report insufficent funds when there are insufficent funds")
		} else {
			t.Errorf("OrderedCoinStorage.Fund: Reported a diferent error than insufficent funds when there are"+
				"insufficent funds: %s", err.Error())
		}
	} else {
		if len(dst) != 0 {
			t.Errorf("OrderedCoinStorage.Fund: returned a list of coins when funds are insufficent: %v", dst)
		}
		if !chng.IsNil() {
			t.Errorf("OrderedCoinStorage.Fund: returned change when funds are insufficent: %v", chng)
		}
	}
}

// Tests that a single coin equal to the correct value returns properly
func TestOrderedCoinStorage_Fund_Single_Exact(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	dst, chng, err := ocs.Fund(69, 1)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: error returned on valid fund: %s", err.Error())
	}

	if len(dst) != 1 {
		t.Errorf("OrderedCoinStorage.Fund: incorrect number fo coins returned to destroy: %v", dst)
	}

	if !chng.IsNil() {
		t.Errorf("OrderedCoinStorage.Fund: change returned on exact transaction: %v", chng)
	}

	if !reflect.DeepEqual(dst[0], cs) {
		t.Errorf("OrderedCoinStorage.Fund: single exact change coin not returned proeprly: "+
			"Expecterd: %v, Recieved: %v", cs, dst[0])
	}
}

// Tests that a multiple coins equal to the correct value returns properly
func TestOrderedCoinStorage_Fund_Multi_Exact(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	cs, err := coin.NewSleeve(69)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs)

	cs2, err := coin.NewSleeve(42)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
	}

	ocs.Add(cs2)

	dst, chng, err := ocs.Fund(111, 2)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: error returned on valid fund: %s", err.Error())
	}

	if len(dst) != 2 {
		t.Errorf("OrderedCoinStorage.Fund: incorrect number of coins returned to destroy: %v", dst)
	}

	if !chng.IsNil() {
		t.Errorf("OrderedCoinStorage.Fund: change returned on exact transaction: %v", chng)
	}

	if len(*ocs.list) != 0 {
		t.Errorf("OrderedCoinStorage.Fund: coins not removed from ordered list: %v", *ocs.list)
	}

	if !reflect.DeepEqual(dst, []coin.Sleeve{cs2, cs}) {
		t.Errorf("OrderedCoinStorage.Fund: exact change coins not returned proeprly: "+
			"Expecterd: %v, Recieved: %v", []coin.Sleeve{cs2, cs}, dst)
	}
}

// Tests that a multiple coins equal to the correct value returns properly when there are other coins
func TestOrderedCoinStorage_Fund_Multi_Exact_Split(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	valueList := []uint64{42, 69, 16}
	var coinList []coin.Sleeve

	for _, v := range valueList {
		cs, err := coin.NewSleeve(v)
		coinList = append(coinList, cs)

		if err != nil {
			t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
		}

		ocs.Add(cs)
	}

	expected := coinList[:2]

	dst, chng, err := ocs.Fund(111, 2)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: error returned on valid fund: %s", err.Error())
	}

	if len(dst) != 2 {
		t.Errorf("OrderedCoinStorage.Fund: incorrect number of coins returned to destroy: %v", dst)
	}

	if !chng.IsNil() {
		t.Errorf("OrderedCoinStorage.Fund: change returned on exact transaction: %v", chng)
	}

	if len(*ocs.list) != 1 {
		t.Errorf("OrderedCoinStorage.Fund: coins not removed from ordered list: %v", *ocs.list)
	}

	if !reflect.DeepEqual(dst, expected) {
		t.Errorf("OrderedCoinStorage.Fund: exact change coins not returned proeprly: "+
			"Expecterd: %v, Recieved: %v", expected, dst)
	}
}

// Tests that fund returns an error when the value cannot be created in the given number of coins
func TestOrderedCoinStorage_Fund_Organization(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	valueList := []uint64{42, 69, 16}
	var coinList []coin.Sleeve

	for _, v := range valueList {
		cs, err := coin.NewSleeve(v)
		coinList = append(coinList, cs)

		if err != nil {
			t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
		}

		ocs.Add(cs)
	}

	_, _, err = ocs.Fund(111, 1)

	if err != ErrInvalidOrganizationOfFunds {
		if err == nil {
			t.Errorf("OrderedCoinStorage.Fund: no error returned on invalid fund orgonization")
		} else {
			t.Errorf("OrderedCoinStorage.Fund: incorrect error returned on invalid fund orgonization: %s", err.Error())
		}
	}
}

// Tests that a multiple coins equal greater than the expected value returns properly when there are other coins
func TestOrderedCoinStorage_Fund_Multi_Exact_Split_Change(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	key := "TestOrderedList"

	ocs, err := CreateOrderedStorage(key, s)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: ordered storage creation failed: %s", err.Error())
	}

	valueList := []uint64{42, 69, 16}
	var coinList []coin.Sleeve

	for _, v := range valueList {
		cs, err := coin.NewSleeve(v)
		coinList = append(coinList, cs)

		if err != nil {
			t.Errorf("OrderedCoinStorage.Fund: sleeve creation failed: %s", err.Error())
		}

		ocs.Add(cs)
	}

	expected := coinList[:2]

	dst, chng, err := ocs.Fund(105, 2)

	if err != nil {
		t.Errorf("OrderedCoinStorage.Fund: error returned on valid fund: %s", err.Error())
	}

	if len(dst) != 2 {
		t.Errorf("OrderedCoinStorage.Fund: incorrect number of coins returned to destroy: %v", dst)
	}

	if chng.IsNil() {
		t.Errorf("OrderedCoinStorage.Fund: change not returned on inexact transaction: %v", chng)
	}

	if chng.Value() != 6 {
		t.Errorf("OrderedCoinStorage.Fund: change not equal to what it should be: Expected: %v; Recieved: %v",
			6, chng.Value())
	}

	if !reflect.DeepEqual(dst, expected) {
		t.Errorf("OrderedCoinStorage.Fund: exact change coins not returned proeprly: "+
			"Expecterd: %v, Recieved: %v", expected, dst)
	}
}

// Shows that Ordered Storage reloads from a stored map properly
func TestOrderedStorage_FileLoading(t *testing.T) {
	globals.LocalStorage = nil

	globals.InitStorage(&globals.DefaultStorage{}, "C:/Users/benger/.privategrity/s.store")
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

	s2, err := user.LoadSession(1)

	if err != nil {
		t.Errorf("session load error: %s", err.Error())
	}

	ocs2, err := CreateOrderedStorage(key, s2)

	if err != nil {
		t.Errorf("CreateOrderedStorage: error returned on valid ordered storage creation: %s", err.Error())
	}

	if ocs2.session != s2 {
		t.Errorf("CreateOrderedStorage: does not point to session correctly")
	}

	if ocs2.value != 10 {
		t.Errorf("CreateOrderedStorage: inital value incorrect: Expected: %v, Recieved: %v", 0, ocs2.value)
	}

	ns2, err := coin.NewSleeve(10)

	if err != nil {
		t.Errorf("CreateOrderedStorage: sleeve creation failed: %s", err.Error())
	}

	*ocs2.list = append(*ocs2.list, ns2)

	s2.StoreSession()

	s3, err := user.LoadSession(1)

	if err != nil {
		t.Errorf("session load error: %s", err.Error())
	}

	ocs3, err := CreateOrderedStorage(key, s3)

	if err != nil {
		t.Errorf("CreateOrderedStorage: error returned on valid ordered storage creation: %s", err.Error())
	}

	if ocs3.session != s3 {
		t.Errorf("CreateOrderedStorage: does not point to session correctly")
	}

	if ocs3.value != 20 {
		t.Errorf("CreateOrderedStorage: inital value incorrect: Expected: %v, Recieved: %v", 0, ocs3.value)
	}

	ns3, err := coin.NewSleeve(10)

	if err != nil {
		t.Errorf("CreateOrderedStorage: sleeve creation failed: %s", err.Error())
	}

	*ocs3.list = append(*ocs3.list, ns3)

	s3.StoreSession()
}
