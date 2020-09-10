////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// nodes.go implements add/remove of nodes from network and node key exchange.

import (
	//	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	//	"gitlab.com/elixxir/comms/client"
	//	"gitlab.com/elixxir/primitives/format"
	//	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/primitives/id"
	//	"sync"
	//	"time"
)

// StartNodeKeyExchange kicks off a worker pool of node key exchange routines
func StartNodeKeyExchange(ctx *context.Context) stoppable.Stoppable {
	stoppers := stoppable.NewMulti("NodeKeyExchangers")
	// keyCh := ctx.Manager.GetNodeKeysCh()
	// for i := 0; i < ctx.GetNumNodeKeyExchangers(); i++ {
	// 	stopper := stoppable.NewSingle("NodeKeyExchange" + i)
	// 	go ExchangeNodeKeys(ctx, keyCh, stopper.Quit())
	// 	stoppers.Add(stopper)
	// }
	return stoppers
}

// ExchangeNodeKeys adds a given node to a client and stores the keys
// exchanged between the client and the node.
func ExchangeNodeKeys(ctx *context.Context, keyCh chan id.ID,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
			// case nid := <-keyCh:
			// 	nodekey := RegisterNode(ctx, nid) // defined elsewhere...
			// 	ctx.GetStorage().SetNodeKey(nid, nodekey)
		}
	}
}

// StartNodeRemover starts node remover worker pool
func StartNodeRemover(ctx *context.Context) stoppable.Stoppable {
	stoppers := stoppable.NewMulti("NodeKeyExchangers")
	// remCh := ctx.GetNetwork().GetNodeRemCh()
	// for i := 0; i < ctx.GetNumNodeRemovers(); i++ {
	// 	stopper := stoppable.NewSingle("NodeKeyExchange" + i)
	// 	go RemoveNode(ctx, remCh, quitCh)
	// 	stoppers.Add(stopper)
	// }
	return stoppers
}

// RemoveNode removes node ids from the client, deleting their keys.
func RemoveNode(ctx *context.Context, remCh chan id.ID,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
			// case nid := <-keyCh:
			// 	ctx.GetStorage().RemoveNodeKey(nid)
		}
	}
}
