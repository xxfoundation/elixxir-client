////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/network/keyExchange"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	"time"
)

// Receive is called by a MessageReceiver routine whenever a new CMIX message
// is available.
func Receive(ctx *Context, m *CMIXMessage) {
	decrypted, err := decrypt(ctx, m) // Returns MessagePart
	if err != nil {
		// Add to error/garbled messages list
		jww.WARN.Errorf("Could not decode message: %+v", err)
		ctx.GetGarbledMesssages().Add(m)
	}

	// Reconstruct the partitioned message
	completeMsg := constructMessageFromPartition(ctx, decrypted) // Returns ClientMessage
	if completeMsg != nil {
		ctx.GetSwitchBoard().Say(completeMsg)
	}
}

// StartMessageReceivers starts a worker pool of message receivers, which listen
// on a channel for messages and run them through processing.
func StartMessageReceivers(ctx *context.Context) Stoppable {
	// We assume receivers channel is set up elsewhere, but note that this
	// would also be a reasonable place under assumption of 1 call to
	// message receivers (would also make sense to .Close it instead of
	// using quit channel, which somewhat simplifies for loop later.
	receiverCh := ctx.GetNetwork().GetMessageReceiverCh()
	for i := 0; i < ctx.GetNumReceivers(); i++ {
		// quitCh created for each thread, add to multistop
		quitCh := make(chan bool)
		go MessageReceiver(ctx, messagesCh, quitCh)
	}

	// Return multistoppable
}

func MessageReceiver(ctx *context.Context, messagesCh chan ClientMessage,
	quitCh chan bool) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case m := <-messagesCh:
			ReceiveMessage(ctx, m) // defined elsewhere...
		}
	}
}
