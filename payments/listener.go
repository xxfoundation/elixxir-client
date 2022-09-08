////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package payments

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/e2e/receive"
)

// Error messages.
const ()

// Name of listener (used for debugging)
const listenerName = "NewPaymentsListener-E2E"

type listener struct {
	m *manager
}

// Hear is called when a payments request message is received.  It builds a
// channel based on the e2e key residue, stores the channel and calls the
// manager receiveCallback
func (l *listener) Hear(msg receive.Message) {
	info := &PaymentInfo{}
	err := json.Unmarshal(msg.Payload, info)
	if err != nil {
		jww.ERROR.Printf("Failed to unmarshal received payment "+
			"message: %+v", err)
	}

	ch, key, err := makeChannelForRequest(msg.KeyResidue, info)
	if err != nil {
		jww.ERROR.Printf("Failed to make channel for received "+
			"request: %+v", err)
	}

	l.m.pendingRequests[info.Id] = &payment{
		sender:            msg.Sender,
		recipient:         l.m.e2e.GetReceptionIdentity().ID,
		status:            Received,
		receiptChannel:    ch,
		receiptChannelKey: key,
		info:              info,
	}

	go l.m.receiveCallback(info.Address, info.Amount, info.Data, info.Id,
		msg.Sender)
}

// Name returns a name used for debugging.
func (l *listener) Name() string {
	return listenerName
}
