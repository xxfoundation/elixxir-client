package keyExchange

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/primitives/switchboard"
)

func Init(ctx *context.Context) stoppable.Stoppable {

	//register the rekey request thread
	rekeyRequestCh := make(chan switchboard.Item, 10)
	ctx.Switchboard.RegisterChannel()

}
