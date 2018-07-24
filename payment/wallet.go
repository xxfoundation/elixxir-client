////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"gitlab.com/privategrity/client/user"
)

const CoinStorageTag = "CoinStorage"
const OutboundRequestsTag = "OutboundRequests"
const InboundRequestsTag = "InboundRequests"
const PendingTransactionsTag = "PendingTransactions"

type Wallet struct {
	coinStorage         *OrderedCoinStorage
	outboundRequests    *TransactionList
	inboundRequests     *TransactionList
	pendingTransactions *TransactionList

	session user.Session
}

func CreateWallet(s user.Session) (*Wallet, error) {

	cs, err := CreateOrderedStorage(CoinStorageTag, s)

	if err != nil {
		return nil, err
	}

	obr, err := CreateTransactionList(OutboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	ibr, err := CreateTransactionList(InboundRequestsTag, s)

	if err != nil {
		return nil, err
	}

	pt, err := CreateTransactionList(PendingTransactionsTag, s)

	if err != nil {
		return nil, err
	}

	w := &Wallet{coinStorage: cs, outboundRequests: obr,
		inboundRequests: ibr, pendingTransactions: pt}

	return w, nil
}
