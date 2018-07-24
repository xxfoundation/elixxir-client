package payment

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/user"
	"testing"
)

// Shows that CreateWallet creates new wallet properly
func TestCreateWallet(t *testing.T) {
	globals.LocalStorage = nil
	globals.InitStorage(&globals.RamStorage{}, "")
	s := user.NewSession(&user.User{1, "test"}, "", []user.NodeKeys{})

	_, err := CreateWallet(s)

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
