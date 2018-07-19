////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/coin"
	"time"
)

type Transaction struct {
	Create  coin.Sleeve
	Destroy []coin.Sleeve
	Change  coin.Sleeve

	Sender    user.ID
	Recipient user.ID

	Description string

	Timestamp time.Time

	Value uint64
}

// FIXME Limit this to one part message (requires message ID revamp for accuracy)
func (t *Transaction) FormatInvoice() (*parse.Message, error) {
	compound := t.Create.Compound()
	invoice := parse.PaymentInvoice{
		Time:        time.Now().Unix(),
		CreatedCoin: compound[:],
		Memo:        t.Description,
	}
	wireRep, err := proto.Marshal(&invoice)
	if err != nil {
		return nil, err
	}

	typedBody := parse.TypedBody{
		Type: parse.Type_PAYMENT_INVOICE,
		Body: wireRep,
	}

	return &parse.Message{
		TypedBody: typedBody,
		Sender:    t.Sender,
		Receiver:  t.Recipient,
		// TODO populate nonce and panic if any outgoing message has none
		Nonce: nil,
	}, nil
}
