////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/coin"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

type Transaction struct {
	Create  coin.Sleeve
	Destroy []coin.Sleeve
	Change  coin.Sleeve

	Sender    *id.User
	Recipient *id.User

	Memo string

	Timestamp time.Time

	Value uint64

	OriginID parse.MessageHash
}

// FIXME Limit this to one part message (requires message ID revamp for accuracy)
// Place the compound coin that's the vessel for payment in the Create sleeve,
// as it's the coin that will be created on the payment bot.
func (t *Transaction) FormatPaymentInvoice() *parse.Message {
	compound := t.Create.Compound()
	invoice := cmixproto.PaymentInvoice{
		Time:        t.Timestamp.Unix(),
		CreatedCoin: compound[:],
		Memo:        t.Memo,
	}
	wireRep, err := proto.Marshal(&invoice)
	if err != nil {
		// This should never happen
		panic("FormatPaymentInvoice: Got error while marshaling invoice: " +
			err.Error())
	}

	typedBody := parse.TypedBody{
		InnerType: int32(cmixproto.Type_PAYMENT_INVOICE),
		Body: wireRep,
	}

	return &parse.Message{
		TypedBody: typedBody,
		// The person who sends the invoice is the one who will receive the
		// money
		Sender:   t.Recipient,
		Receiver: t.Sender,
		// TODO populate nonce and panic if any outgoing message has none
		Nonce: nil,
	}
}
