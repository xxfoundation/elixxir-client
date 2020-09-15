////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/primitives/format"
)

// ReceiveMessage is called by a MessageReceiver routine whenever a new CMIX
// message is available.
func ReceiveMessage(ctx *context.Context, m *format.Message) {
	// decrypted, err := decrypt(ctx, m) // Returns MessagePart
	// if err != nil {
	// 	// Add to error/garbled messages list
	// 	jww.WARN.Errorf("Could not decode message: %+v", err)
	// 	ctx.GetGarbledMesssages().Add(m)
	// }

	// // Reconstruct the partitioned message
	// completeMsg := constructMessageFromPartition(ctx, decrypted) // Returns ClientMessage
	// if completeMsg != nil {
	// 	ctx.GetSwitchBoard().Say(completeMsg)
	// }
}

// StartMessageReceivers starts a worker pool of message receivers, which listen
// on a channel for messages and run them through processing.
func StartMessageReceivers(ctx *context.Context) stoppable.Stoppable {
	// We assume receivers channel is set up elsewhere, but note that this
	// would also be a reasonable place under assumption of 1 call to
	// message receivers (would also make sense to .Close it instead of
	// using quit channel, which somewhat simplifies for loop later.
	stoppers := stoppable.NewMulti("MessageReceivers")
	// receiverCh := ctx.GetNetwork().GetMessageReceiverCh()
	// for i := 0; i < ctx.GetNumReceivers(); i++ {
	// 	stopper := stoppable.NewSingle("MessageReceiver" + i)
	// 	go MessageReceiver(ctx, messagesCh, stopper.Quit())
	// 	stoppers.Add(stopper)
	// }
	return stoppers
}

// MessageReceiver waits until quit signal or there is a message
// available on the messages channel.
func MessageReceiver(ctx *context.Context, messagesCh chan format.Message,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
			// case m := <-messagesCh:
			// 	ReceiveMessage(ctx, m)
		}
	}
}
