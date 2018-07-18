package payment

const CoinStorageTag string = "CoinStorage"
const OutboundRequestsTag string = "OutboundRequests"
const InboundRequestsTag string = "InboundRequests"
const PendingTransactionsTag string = "PendingTransactions"

type Wallet struct {
	coinStorage 			*OrderedCoinStorage
	outboundRequests 		*TransactionList
	inboundRequests  		*TransactionList
	pendingTransactions 	*TransactionList
}

func NewWallet()(*Wallet,error){

	cs, err := NewOrderedStorage(CoinStorageTag)

	if err!=nil{
		return nil,err
	}

	obr, err := NewTransactionList(OutboundRequestsTag)

	if err!=nil{
		return nil,err
	}

	ibr, err := NewTransactionList(InboundRequestsTag)

	if err!=nil{
		return nil,err
	}

	pt, err := NewTransactionList(PendingTransactionsTag)

	if err!=nil{
		return nil,err
	}

	return &Wallet{coinStorage:cs, outboundRequests: obr, inboundRequests: ibr, pendingTransactions:pt}, nil
}

