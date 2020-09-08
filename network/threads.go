////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// threads.go handles all the long running network processing threads in client

import (
	"gitlab.com/elixxir/client/context"
	"time"
)

// Interface for stopping a goroutine
type Stoppable interface {
	Close(timeout time.Duration)
}

// ChanStop allows stopping a single goroutine using a channel
type ChanStop struct {
	name string
	quit chan bool
}

// Close signals thread to time out and closes.
func (c ChanStop) Close(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	select {
	case <-timer:
		jww.ERROR.Printf("goroutine failed to Close: %s", c.name)
	case <-c.quit:
		return
	}
}

// StartTrackNetwork starts a single TrackNetwork thread and returns a stoppable
// structure
func StartTrackNetwork(ctx *context.Context) Stoppable {
	stopper := ChanStop{
		name: "TrackNetwork",
		quit: make(chan bool),
	}
	go TrackNetwork(ctx, stopper.quit)
	return stopper
}

// TrackNetwork polls the network to get updated on the state of nodes, the
// round status, and informs the client when messages can be retrieved.
func TrackNetwork(ctx *context.Context, quitCh chan bool) {
	ticker := timer.NewTicker(ctx.GetTrackNetworkPeriod())
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case <-ticker:
			trackNetwork(ctx)
		}
	}
}

func trackNetwork(ctx) {
	gateway, err := ctx.Session.GetNodeKeys().GetGatewayForSending()
	if err != nil {
		//...
	}

	network := ctx.GetNetwork()
	ndf, err := network.PollNDF(ctx, gateway)
	if err != nil {
		// ....
	}

	newNodes, removedNodes := network.UpdateNDF(ndf)
	for _, n := range newNodes {
		network.addNodeCh <- n
	}
	for _, n := range removedNodes {
		network.removeNodeCh <- n
	}

	rounds, err = network.UpdateRounds(ctx, ndf)
	if err != nil {
		// ...
	}

	err = rounds.GetKnownRound().MaskedRange(gateway,
		network.CheckRoundsFunction)
	if err != nil {
		// ...
	}
}

func StartProcessHistoricalRounds(ctx *context.Context) Stoppable {
	stopper := ChanStop{
		name: "ProcessHistoricalRounds",
		quit: make(chan bool),
	}
	go ProcessHistoricalRounds(ctx, stopper.quit)
	return stopper
}

func ProcessHistoricalRounds(ctx *context.Context, quitCh chan bool) {
	ticker := timer.NewTicker(ctx.GetTrackNetworkPeriod())
	var rounds []RoundID
	done := false
	for !done {
		shouldProcess := false
		select {
		case <-quitCh:
			done = true
		case <-ticker:
			if len(rounds) > 0 {
				shouldProcess = true
			}
		case rid := <-ctx.GetHistoricalRoundsCh():
			rounds = append(rounds, rid)
			if len(rounds) > ctx.GetSendSize() {
				shouldProcess = true
			}
		}
		if !shouldProcess {
			continue
		}

		var roundInfos []*RoundInfo
		roundInfos := processHistoricalRounds(ctx, rounds)
		rounds := make([]RoundID)
		for _, ri := range roundInfos {
			ctx.GetMessagesCh() <- ri
		}
	}
}

func processHistoricalRounds(ctx *context.Context, rids []RoundID) []*RoundInfo {
	// for loop over rids?
	network := ctx.GetNetwork()
	gw := network.GetGateway()
	ris := gw.GetHistoricalRounds(ctx.GetRoundList())
	return ris
}

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

func StartNodeKeyExchange(ctx *context.Context) {
	keyCh := ctx.GetNetwork().GetNodeKeysCh()
	for i := 0; i < ctx.GetNumNodeKeyExchangers(); i++ {
		// quitCh created for each thread, add to multistop
		quitCh := make(chan bool)
		go ExchangeNodeKeys(ctx, keyCh, quitCh)
	}

	// return multistoppable
}

func ExchangeNodeKeys(ctx *context.Context, keyCh chan node.ID, quitCh chan bool) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case nid := <-keyCh:
			nodekey := RegisterNode(ctx, nid) // defined elsewhere...
			ctx.GetStorage().SetNodeKey(nid, nodekey)
		}
	}
}

func StartNodeRemover(ctx *context.Context) {
	remCh := ctx.GetNetwork().GetNodeRemCh()
	for i := 0; i < ctx.GetNumNodeRemovers(); i++ {
		// quitCh created for each thread, add to multistop
		quitCh := make(chan bool)
		go RemoveNode(ctx, remCh, quitCh)
	}

	// return multistoppable
}

func RemoveNode(ctx *context.Context, remCh chan node.ID, quitCh chan bool) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case nid := <-keyCh:
			ctx.GetStorage().RemoveNodeKey(nid)
		}
	}
}
