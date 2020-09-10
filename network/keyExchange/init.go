package keyExchange

import (
	"github.com/golang/protobuf/proto"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/storage/e2e"
	"gitlab.com/xx_network/primitives/id"
)

const keyExchangeTriggerName = "KeyExchangeTrigger"

func Init(ctx *context.Context) stoppable.Stoppable {

	//register the rekey request thread
	rekeyRequestCh := make(chan message.Receive, 100)
	ctx.Switchboard.RegisterChannel(keyExchangeTriggerName, &id.ID{},
		message.KeyExchangeTrigger, rekeyRequestCh)

	triggerStop := stoppable.NewSingle(keyExchangeTriggerName)

}


