////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

import (
	"fmt"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/format"
	//jww "github.com/spf13/jwalterweatherman"
)

// ReceiveMessages is called by a MessageReceiver routine whenever new CMIX
// messages are available at a gateway.
func ReceiveMessages(ctx *context.Context, roundInfo *pb.RoundInfo) {
	msgs := getMessagesFromGateway(ctx, roundInfo)
	for _, m := range msgs {
		receiveMessage(ctx, m)
	}
}

// StartMessageReceivers starts a worker pool of message receivers, which listen
// on a channel for rounds in which to check for messages and run them through
// processing.
func StartMessageReceivers(ctx *context.Context,
	network *Manager) stoppable.Stoppable {
	stoppers := stoppable.NewMulti("MessageReceivers")
	opts := params.GetDefaultNetwork()
	receiverCh := network.GetRoundUpdateCh()
	for i := 0; i < opts.NumWorkers; i++ {
		stopper := stoppable.NewSingle(
			fmt.Sprintf("MessageReceiver%d", i))
		go MessageReceiver(ctx, receiverCh, stopper.Quit())
		stoppers.Add(stopper)
	}
	return stoppers
}

// MessageReceiver waits until quit signal or there is a round available
// for which to check for messages available on the round updates channel.
func MessageReceiver(ctx *context.Context, updatesCh chan *pb.RoundInfo,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case round := <-updatesCh:
			ReceiveMessages(ctx, round)
		}
	}
}

func getMessagesFromGateway(ctx *context.Context,
	roundInfo *pb.RoundInfo) []format.Message {
	return nil
}

func receiveMessage(ctx *context.Context, msg format.Message) {
	// do stuff
}
