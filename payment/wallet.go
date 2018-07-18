package payment

const CoinStorageTag string = "CoinStorage"

type Wallet struct {
	coinStorage *OrderedStorage
}

func NewWallet()(*Wallet,error){

	cs, err := NewOrderedStorage(CoinStorageTag)

	if err!=nil{
		return nil,err
	}

	return &Wallet{coinStorage:cs}, nil
}

