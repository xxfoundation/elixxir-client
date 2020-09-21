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
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/network/rounds"

	//	"gitlab.com/elixxir/comms/client"
	//	"gitlab.com/elixxir/primitives/format"
	//	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/primitives/id"
	//	"sync"
	//	"time"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
)

// StartNodeKeyExchange kicks off a worker pool of node key exchange routines
func StartNodeKeyExchange(ctx *context.Context,
	network *rounds.Manager) stoppable.Stoppable {
	stoppers := stoppable.NewMulti("NodeKeyExchangers")
	numWorkers := params.GetDefaultNetwork().NumWorkers
	keyCh := network.GetNodeRegistrationCh()
	ctx.Manager.GetInstance().SetAddNodeChan(keyCh)
	for i := 0; i < numWorkers; i++ {
		stopper := stoppable.NewSingle(
			fmt.Sprintf("NodeKeyExchange%d", i))
		go ExchangeNodeKeys(ctx, keyCh, stopper.Quit())
		stoppers.Add(stopper)
	}
	return stoppers
}

// ExchangeNodeKeys adds a given node to a client and stores the keys
// exchanged between the client and the node.
func ExchangeNodeKeys(ctx *context.Context, keyCh chan network.NodeGateway,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case nid := <-keyCh:
			jww.ERROR.Printf("Unimplemented ExchangeNodeKeys: %+v",
				nid)
			// 	nodekey := RegisterNode(ctx, nid) // defined elsewhere...
			// 	ctx.GetStorage().SetNodeKey(nid, nodekey)
		}
	}
}

// StartNodeRemover starts node remover worker pool
func StartNodeRemover(ctx *context.Context) stoppable.Stoppable {
	stoppers := stoppable.NewMulti("NodeKeyExchangers")
	numWorkers := params.GetDefaultNodeKeys().WorkerPoolSize
	remCh := make(chan *id.ID, numWorkers)
	ctx.Manager.GetInstance().SetRemoveNodeChan(remCh)
	for i := uint(0); i < numWorkers; i++ {
		stopper := stoppable.NewSingle(
			fmt.Sprintf("RemoveNode%d", i))
		go RemoveNode(ctx, remCh, stopper.Quit())
		stoppers.Add(stopper)
	}
	return stoppers
}

// RemoveNode removes node ids from the client, deleting their keys.
func RemoveNode(ctx *context.Context, remCh chan *id.ID,
	quitCh <-chan struct{}) {
	done := false
	for !done {
		select {
		case <-quitCh:
			done = true
		case nid := <-remCh:
			jww.ERROR.Printf("Unimplemented RemoveNode: %+v",
				nid)
			// 	ctx.GetStorage().RemoveNodeKey(nid)
		}
	}
}
