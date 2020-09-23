package keyExchange

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

const keyExchangeTriggerName = "KeyExchangeTrigger"
const keyExchangeConfirmName = "KeyExchangeConfirm"
const keyExchangeMulti = "KeyExchange"

func Start(switchboard *switchboard.Switchboard, sess *storage.Session,
	net context.NetworkManager, garbledMessageTrigger chan<- struct{}) stoppable.Stoppable {

	// register the rekey trigger thread
	triggerCh := make(chan message.Receive, 100)
	triggerID := switchboard.RegisterChannel(keyExchangeTriggerName,
		&id.ID{}, message.KeyExchangeTrigger, triggerCh)

	// create the trigger stoppable
	triggerStop := stoppable.NewSingle(keyExchangeTriggerName)
	triggerStopCleanup := stoppable.NewCleanup(triggerStop,
		func(duration time.Duration) error {
			switchboard.Unregister(triggerID)
			return nil
		})

	// start the trigger thread
	go startTrigger(sess, net, triggerCh, triggerStop, garbledMessageTrigger)

	//register the rekey confirm thread
	confirmCh := make(chan message.Receive, 100)
	confirmID := switchboard.RegisterChannel(keyExchangeConfirmName,
		&id.ID{}, message.KeyExchangeConfirm, confirmCh)

	// register the confirm stoppable
	confirmStop := stoppable.NewSingle(keyExchangeConfirmName)
	confirmStopCleanup := stoppable.NewCleanup(confirmStop,
		func(duration time.Duration) error {
			switchboard.Unregister(confirmID)
			return nil
		})

	// start the confirm thread
	go startConfirm(sess, confirmCh, confirmStop)

	//bundle the stoppables and return
	exchangeStop := stoppable.NewMulti(keyExchangeMulti)
	exchangeStop.Add(triggerStopCleanup)
	exchangeStop.Add(confirmStopCleanup)
	return exchangeStop
}
