////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// nodes.go implements add/remove of nodes from network and node key exchange.

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

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
