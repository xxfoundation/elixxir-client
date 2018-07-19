package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"math/rand"
	"reflect"
	"testing"
)

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

func TestOrderedCoinStorage_Value(t *testing.T) {

	src := rand.NewSource(42)
	rng := rand.New(src)

	for i := 0; i < 100; i++ {
		value := rng.Uint64() % uint64(coin.MaxValueDenominationRegister)

		ocs := &OrderedCoinStorage{nil, value, nil}

		if ocs.Value() != value {
			t.Errorf("OrderedCoinStorage.Value: Returned incorrect value: "+
				"Expected: %v, recieved: %v", value, ocs.value)
		}

	}
}
