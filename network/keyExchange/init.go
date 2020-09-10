package keyExchange

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/xx_network/primitives/id"
)

const keyExchangeTriggerName = "KeyExchangeTrigger"

func Init(ctx *context.Context) stoppable.Stoppable {

	//register the rekey request thread
	rekeyRequestCh := make(chan message.Receive, 10)
	ctx.Switchboard.RegisterChannel(keyExchangeTriggerName, &id.ID{},
		message.KeyExchangeTrigger, rekeyRequestCh)

	triggerStop := stoppable.NewSingle(keyExchangeTriggerName)

	go func() {
		for true {
			select {
			case <-triggerStop.Quit():
				return
				//			case request := <-rekeyRequestCh:
				//				return
				//ctx.Session.request.Sender
			}
		}
	}()

	return triggerStop
}
