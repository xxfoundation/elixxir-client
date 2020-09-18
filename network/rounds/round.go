package rounds

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/storage"
)

func StartProcessors(ctx *context.Context) stoppable.Stoppable {
	p := newProcessingRounds()
	stopper := stoppable.NewSingle("TrackNetwork")
	go trackNetwork(ctx, net, stopper.Quit())
}
